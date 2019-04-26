// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

var (
	ErrMakeBlockTransferFail  = fault.ProcessError("make block transfer failed")
	ErrNotBlockTransferRecord = fault.InvalidError("not block transfer record")
)

// BlockTransferData - data for a block transfer request
type BlockTransferData struct {
	Payments currency.Map
	Owner    *keypair.KeyPair
	NewOwner *keypair.KeyPair
	TxId     string
}

// BlockTransferReply - JSON data to output after blockTransfer completes
type BlockTransferReply struct {
	BlockTransferId merkle.Digest                                   `json:"blockTransferId"`
	PayId           pay.PayId                                       `json:"payId"`
	Payments        map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands        map[string]string                               `json:"commands,omitempty"`
}

// BlockTransferSingleSignedReply - reply if performing a single signature transfer
type BlockTransferSingleSignedReply struct {
	Identity      string `json:"identity"`
	BlockTransfer string `json:"blockTransfer"`
}

// SingleSignedBlockTransfer - perform a transfer
func (client *Client) SingleSignedBlockTransfer(blockTransferConfig *BlockTransferData) (*BlockTransferSingleSignedReply, error) {

	var link merkle.Digest
	err := link.UnmarshalText([]byte(blockTransferConfig.TxId))
	if nil != err {
		return nil, err
	}

	packed, blockTransfer, err := makeBlockTransferOneSignature(client.testnet, link, blockTransferConfig.Payments, blockTransferConfig.Owner, blockTransferConfig.NewOwner)
	if nil != err {
		return nil, err
	}
	if nil == blockTransfer {
		return nil, ErrMakeBlockTransferFail
	}

	client.printJson("BlockTransfer Request", blockTransfer)

	response := BlockTransferSingleSignedReply{
		Identity:      blockTransfer.GetOwner().String(),
		BlockTransfer: hex.EncodeToString(packed),
	}

	return &response, nil
}

// CountersignBlockTransfer - perform a transfer
func (client *Client) CountersignBlockTransfer(blockTransfer *transactionrecord.BlockOwnerTransfer) (*BlockTransferReply, error) {

	var reply rpc.BlockOwnerTransferReply
	err := client.client.Call("BlockOwner.Transfer", blockTransfer, &reply)
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

	client.printJson("BlockTransfer Reply", reply)

	// make response
	response := BlockTransferReply{
		BlockTransferId: reply.TxId,
		PayId:           reply.PayId,
		Payments:        reply.Payments,
		Commands:        commands,
	}

	return &response, nil
}

func makeBlockTransferOneSignature(testnet bool, link merkle.Digest, payments currency.Map, owner *keypair.KeyPair, newOwner *keypair.KeyPair) ([]byte, *transactionrecord.BlockOwnerTransfer, error) {

	newOwnerAddress := makeAddress(newOwner, testnet)
	r := transactionrecord.BlockOwnerTransfer{
		Link:             link,
		Version:          1,
		Payments:         payments,
		Owner:            newOwnerAddress,
		Signature:        nil,
		Countersignature: nil,
	}

	ownerAddress := makeAddress(owner, testnet)

	// pack without signature
	packed, err := r.Pack(ownerAddress)
	if nil == err {
		return nil, nil, ErrMakeBlockTransferFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	signature := ed25519.Sign(owner.PrivateKey, packed)
	r.Signature = signature[:]

	// include first signature by packing again
	packed, err = r.Pack(ownerAddress)
	if nil == err {
		return nil, nil, ErrMakeBlockTransferFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
