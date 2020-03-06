// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"github.com/bitmark-inc/bitmarkd/rpc/blockowner"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// BlockTransferData - data for a block transfer request
type BlockTransferData struct {
	Payments currency.Map
	Owner    *configuration.Private
	NewOwner *account.Account
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
		return nil, fault.MakeBlockTransferFailed
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

	var reply blockowner.TransferReply
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
		c := payment[0].Currency
		commands[c.String()] = paymentCommand(client.testnet, c, string(tpid), payment)
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

func makeBlockTransferOneSignature(_ bool, link merkle.Digest, payments currency.Map, owner *configuration.Private, newOwner *account.Account) ([]byte, *transactionrecord.BlockOwnerTransfer, error) {

	r := transactionrecord.BlockOwnerTransfer{
		Link:             link,
		Version:          1,
		Payments:         payments,
		Owner:            newOwner,
		Signature:        nil,
		Countersignature: nil,
	}

	ownerAccount := owner.PrivateKey.Account()

	// pack without signature
	packed, err := r.Pack(ownerAccount)
	if nil == err {
		return nil, nil, fault.MakeBlockTransferFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	signature := ed25519.Sign(owner.PrivateKey.PrivateKeyBytes(), packed)
	r.Signature = signature[:]

	// include first signature by packing again
	packed, err = r.Pack(ownerAccount)
	if nil == err {
		return nil, nil, fault.MakeBlockTransferFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
