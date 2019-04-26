// Copyright (c) 2014-2019 Bitmark Inc.
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
	ErrMakeTransferFail  = fault.ProcessError("make transfer failed")
	ErrNotTransferRecord = fault.InvalidError("not transfer record")
)

// TransferData - data for a transfer request
type TransferData struct {
	Owner    *keypair.KeyPair
	NewOwner *keypair.KeyPair
	TxId     string
}

// TransferCountersignData - countersign data request
type TransferCountersignData struct {
	Transfer string
	NewOwner *keypair.KeyPair
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
		return nil, ErrMakeTransferFail
	}

	client.printJson("Transfer Request", transfer)

	var reply rpc.BitmarkTransferReply
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
		return nil, ErrMakeTransferFail
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

	var reply rpc.BitmarkTransferReply
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

func makeTransferUnratified(testnet bool, link merkle.Digest, owner *keypair.KeyPair, newOwner *keypair.KeyPair) (transactionrecord.BitmarkTransfer, error) {

	newOwnerAddress := makeAddress(newOwner, testnet)
	r := transactionrecord.BitmarkTransferUnratified{
		Link:      link,
		Owner:     newOwnerAddress,
		Signature: nil,
	}

	ownerAddress := makeAddress(owner, testnet)

	// pack without signature
	packed, err := r.Pack(ownerAddress)
	if nil == err {
		return nil, ErrMakeTransferFail
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

func makeTransferOneSignature(testnet bool, link merkle.Digest, owner *keypair.KeyPair, newOwner *keypair.KeyPair) ([]byte, transactionrecord.BitmarkTransfer, error) {

	newOwnerAddress := makeAddress(newOwner, testnet)
	r := transactionrecord.BitmarkTransferCountersigned{
		Link:             link,
		Owner:            newOwnerAddress,
		Signature:        nil,
		Countersignature: nil,
	}

	ownerAddress := makeAddress(owner, testnet)

	// pack without signature
	packed, err := r.Pack(ownerAddress)
	if nil == err {
		return nil, nil, ErrMakeTransferFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}

	// attach signature
	signature := ed25519.Sign(owner.PrivateKey, packed)
	r.Signature = signature[:]

	// include first signature by packing again
	packed, err = r.Pack(ownerAddress)
	if nil == err {
		return nil, nil, ErrMakeTransferFail
	} else if fault.ErrInvalidSignature != err {
		return nil, nil, err
	}
	return packed, &r, nil
}
