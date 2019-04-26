// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

var (
	ErrMakeShareFail = fault.ProcessError("make share failed")
)

// ShareData - data for a share request
type ShareData struct {
	Owner    *keypair.KeyPair
	NewOwner *keypair.KeyPair
	TxId     string
	Quantity uint64
}

// ShareReply - JSON data to output after transfer completes
type ShareReply struct {
	TxId     merkle.Digest                                   `json:"txId"`
	ShareId  merkle.Digest                                   `json:"shareId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands map[string]string                               `json:"commands,omitempty"`
}

// Share - perform a share request
func (client *Client) Share(shareConfig *ShareData) (*ShareReply, error) {

	var link merkle.Digest
	err := link.UnmarshalText([]byte(shareConfig.TxId))
	if nil != err {
		return nil, err
	}

	share, err := makeShare(client.testnet, link, shareConfig.Quantity, shareConfig.Owner)
	if nil != err {
		return nil, err
	}
	if nil == share {
		return nil, ErrMakeShareFail
	}

	client.printJson("Share Request", share)

	var reply rpc.ShareCreateReply
	err = client.client.Call("Share.Create", share, &reply)
	if err != nil {
		return nil, err
	}

	tpid, err := reply.PayId.MarshalText()
	if nil != err {
		return nil, err
	}

	commands := make(map[string]string)
	for _, payment := range reply.Payments {
		currency := payment[0].Currency
		commands[currency.String()] = paymentCommand(client.testnet, currency, string(tpid), payment)
	}

	client.printJson("Share Reply", reply)

	// make response
	response := ShareReply{
		TxId:     reply.TxId,
		ShareId:  reply.ShareId,
		PayId:    reply.PayId,
		Payments: reply.Payments,
		Commands: commands,
	}

	return &response, nil
}

func makeShare(testnet bool, link merkle.Digest, quantity uint64, owner *keypair.KeyPair) (*transactionrecord.BitmarkShare, error) {

	r := transactionrecord.BitmarkShare{
		Link:      link,
		Quantity:  quantity,
		Signature: nil,
	}

	ownerAddress := makeAddress(owner, testnet)

	// pack without signature
	packed, err := r.Pack(ownerAddress)
	if nil == err {
		return nil, ErrMakeShareFail
	} else if fault.ErrInvalidSignature != err {
		return nil, err
	}

	// attach signature
	signature := ed25519.Sign(owner.PrivateKey, packed)
	r.Signature = signature[:]

	// check that signature is correct by packing again
	_, err = r.Pack(ownerAddress)
	if nil != err {
		return nil, err
	}
	return &r, nil
}
