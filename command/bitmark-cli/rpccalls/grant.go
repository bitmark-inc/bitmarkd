// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

var (
	ErrMakeGrantFail  = fault.ProcessError("make grant failed")
	ErrNotGrantRecord = fault.InvalidError("not grant record")
)

type GrantData struct {
	ShareId     string
	Quantity    uint64
	Owner       *keypair.KeyPair
	Recipient   *keypair.KeyPair
	BeforeBlock uint64
}

type GrantCountersignData struct {
	Grant     string
	Recipient *keypair.KeyPair
}

// JSON data to output after grant completes
type GrantReply struct {
	GrantId  merkle.Digest                                   `json:"grantId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands map[string]string                               `json:"commands,omitempty"`
}

type GrantSingleSignedReply struct {
	Identity string `json:"identity"`
	Grant    string `json:"grant"`
}

func (client *Client) Grant(grantConfig *GrantData) (*GrantSingleSignedReply, error) {

	var shareId merkle.Digest
	err := shareId.UnmarshalText([]byte(grantConfig.ShareId))
	if nil != err {
		return nil, err
	}

	if 0 == grantConfig.BeforeBlock {
		info, err := client.GetBitmarkInfo()
		if nil != err {
			return nil, err
		}
		grantConfig.BeforeBlock = info.Blocks.Height + 100 // allow plenty of time to mine
	}

	packed, grant, err := makeGrantOneSignature(client.testnet, shareId, grantConfig.Quantity, grantConfig.Owner, grantConfig.Recipient, grantConfig.BeforeBlock)
	if nil != err {
		return nil, err
	}
	if nil == grant {
		return nil, ErrMakeGrantFail
	}

	client.printJson("Grant Request", grant)

	response := GrantSingleSignedReply{
		Identity: grant.Owner.String(),
		Grant:    hex.EncodeToString(packed),
	}

	return &response, nil
}

func (client *Client) CountersignGrant(grant *transactionrecord.ShareGrant) (*GrantReply, error) {

	client.printJson("Grant Request", grant)

	var reply rpc.ShareGrantReply
	err := client.client.Call("Share.Grant", grant, &reply)
	if nil != err {
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

	client.printJson("Grant Reply", reply)

	// make response
	response := GrantReply{
		GrantId:  reply.TxId,
		PayId:    reply.PayId,
		Payments: reply.Payments,
		Commands: commands,
	}

	return &response, nil
}

func makeGrantOneSignature(testnet bool, shareId merkle.Digest, quantity uint64, owner *keypair.KeyPair, recipient *keypair.KeyPair, beforeBlock uint64) ([]byte, *transactionrecord.ShareGrant, error) {

	ownerAddress := makeAddress(owner, testnet)
	recipientAddress := makeAddress(recipient, testnet)

	r := transactionrecord.ShareGrant{
		ShareId:          shareId,
		Quantity:         quantity,
		Owner:            ownerAddress,
		Recipient:        recipientAddress,
		BeforeBlock:      beforeBlock,
		Signature:        nil,
		Countersignature: nil,
	}

	// pack without signature
	packed, err := r.Pack(ownerAddress)
	if nil == err {
		return nil, nil, ErrMakeGrantFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	signature := ed25519.Sign(owner.PrivateKey, packed)
	r.Signature = signature[:]

	// include first signature by packing again
	packed, err = r.Pack(ownerAddress)
	if nil == err {
		return nil, nil, ErrMakeGrantFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
