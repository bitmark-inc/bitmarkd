// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"encoding/hex"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/rpc/bitmark"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// TransferData - data for a transfer request
type TransferData struct {
	Owner    *configuration.Private
	NewOwner *account.Account
	TxId     string
}

// TransferCountersignData - countersign data request
type TransferCountersignData struct {
	Transfer string
	NewOwner *configuration.Private
}

// TransferReply - JSON data to output after transfer completes
type TransferReply struct {
	TransferId merkle.Digest                                   `json:"transferId"`
	BitmarkId  merkle.Digest                                   `json:"bitmarkId"`
	PayId      pay.PayId                                       `json:"payId"`
	Payments   map[string]transactionrecord.PaymentAlternative `json:"payments"`
	Commands   map[string]string                               `json:"commands,omitempty"`
}

// TransferSingleSignedReply - response to single signature
type TransferSingleSignedReply struct {
	Identity string `json:"identity"`
	Transfer string `json:"transfer"`
}

// Transfer - perform a bitmark transfer
func (client *Client) Transfer(transferConfig *TransferData) (*TransferReply, error) {

	var link merkle.Digest
	err := link.UnmarshalText([]byte(transferConfig.TxId))
	if nil != err {
		return nil, err
	}

	transfer, err := makeTransferUnratified(client.testnet, link, transferConfig.Owner, transferConfig.NewOwner)
	if nil != err {
		return nil, err
	}
	if nil == transfer {
		return nil, fault.MakeTransferFailed
	}

	client.printJson("Transfer Request", transfer)

	var reply bitmark.TransferReply
	err = client.client.Call("Bitmark.Transfer", transfer, &reply)
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

	client.printJson("Transfer Reply", reply)

	// make response
	response := TransferReply{
		TransferId: reply.TxId,
		BitmarkId:  reply.BitmarkId,
		PayId:      reply.PayId,
		Payments:   reply.Payments,
		Commands:   commands,
	}

	return &response, nil
}

// SingleSignedTransfer - perform a single signed transfer
func (client *Client) SingleSignedTransfer(transferConfig *TransferData) (*TransferSingleSignedReply, error) {

	var link merkle.Digest
	err := link.UnmarshalText([]byte(transferConfig.TxId))
	if nil != err {
		return nil, err
	}

	packed, transfer, err := makeTransferOneSignature(client.testnet, link, transferConfig.Owner, transferConfig.NewOwner)
	if nil != err {
		return nil, err
	}
	if nil == transfer {
		return nil, fault.MakeTransferFailed
	}

	client.printJson("Transfer Request", transfer)

	response := TransferSingleSignedReply{
		Identity: transfer.GetOwner().String(),
		Transfer: hex.EncodeToString(packed),
	}

	return &response, nil
}

// CountersignTransfer - perform as countersigned transfer
func (client *Client) CountersignTransfer(transfer *transactionrecord.BitmarkTransferCountersigned) (*TransferReply, error) {

	client.printJson("Transfer Request", transfer)

	var reply bitmark.TransferReply
	err := client.client.Call("Bitmark.Transfer", transfer, &reply)
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

	client.printJson("Transfer Reply", reply)

	// make response
	response := TransferReply{
		TransferId: reply.TxId,
		BitmarkId:  reply.BitmarkId,
		PayId:      reply.PayId,
		Payments:   reply.Payments,
		Commands:   commands,
	}

	return &response, nil
}

func makeTransferUnratified(testnet bool, link merkle.Digest, owner *configuration.Private, newOwner *account.Account) (transactionrecord.BitmarkTransfer, error) {

	r := transactionrecord.BitmarkTransferUnratified{
		Link:      link,
		Owner:     newOwner,
		Signature: nil,
	}

	ownerAccount := owner.PrivateKey.Account()

	// pack without signature
	packed, err := r.Pack(ownerAccount)
	if nil == err {
		return nil, fault.MakeTransferFailed
	} else if fault.InvalidSignature != err {
		return nil, err
	}

	// attach signature
	signature := ed25519.Sign(owner.PrivateKey.PrivateKeyBytes(), packed)
	r.Signature = signature[:]

	// check that signature is correct by packing again
	_, err = r.Pack(ownerAccount)
	if nil != err {
		return nil, err
	}
	return &r, nil
}

func makeTransferOneSignature(testnet bool, link merkle.Digest, owner *configuration.Private, newOwner *account.Account) ([]byte, transactionrecord.BitmarkTransfer, error) {

	r := transactionrecord.BitmarkTransferCountersigned{
		Link:             link,
		Owner:            newOwner,
		Signature:        nil,
		Countersignature: nil,
	}

	ownerAccount := owner.PrivateKey.Account()

	// pack without signature
	packed, err := r.Pack(ownerAccount)
	if nil == err {
		return nil, nil, fault.MakeTransferFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	signature := ed25519.Sign(owner.PrivateKey.PrivateKeyBytes(), packed)
	r.Signature = signature[:]

	// include first signature by packing again
	packed, err = r.Pack(ownerAccount)
	if nil == err {
		return nil, nil, fault.MakeTransferFailed
	} else if fault.InvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
