// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/currency" // ***** FIX THIS: remove when real currency/address is available
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Bitmarks
// --------

type Bitmarks struct {
	log *logger.L
}

const (
	maximumIssues = 100
)

// Bitmarks issue
// --------------

type IssueStatus struct {
	TxId merkle.Digest `json:"txId"`
}

type BitmarksIssueReply struct {
	Issues     []IssueStatus    `json:"issues"`
	PayId      payment.PayId    `json:"payId"`
	PayNonce   payment.PayNonce `json:"payNonce"`
	Difficulty string           `json:"difficulty"`
	//PaymentAlternatives []block.MinerAddress `json:"paymentAlternatives"`// ***** FIX THIS: where to get addresses?
}

func (bitmarks *Bitmarks) Issue(arguments *[]transactionrecord.BitmarkIssue, reply *BitmarksIssueReply) error {

	log := bitmarks.log
	count := len(*arguments)

	if count > maximumIssues {
		return fault.ErrTooManyItemsToProcess
	} else if 0 == count {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	log.Infof("Bitmarks.Issue: %v", arguments)

	issueStatus, packed, err := bitmarksIssue(*arguments)
	if nil != err {
		return err
	}

	result := BitmarksIssueReply{
		Issues: issueStatus,
	}

	// fail if no data sent
	if 0 == len(packed) {
		return fault.ErrMissingParameters
	}

	// get here if all issues are new
	var d *difficulty.Difficulty
	newItem := false
	result.PayId, result.PayNonce, d, newItem = payment.Store(currency.Bitcoin, packed, count, true)
	result.Difficulty = d.GoString()

	// announce transaction block to other peers
	if newItem {
		messagebus.Bus.Broadcast.Send("issues", packed)
	}

	*reply = result
	return nil
}

// internal function to issue some bitmarks
func bitmarksIssue(issues []transactionrecord.BitmarkIssue) ([]IssueStatus, []byte, error) {

	issueStatus := make([]IssueStatus, len(issues))

	// pack each transaction
	packed := []byte{}
	for i, argument := range issues {

		packedIssue, err := argument.Pack(argument.Owner)
		if nil != err {
			return nil, nil, err
		}

		if !asset.Exists(argument.AssetIndex) {
			return nil, nil, fault.ErrAssetNotFound
		}

		txId := packedIssue.MakeLink()
		issueStatus[i].TxId = txId
		key := txId[:]

		// even a single verified/confirmed issue fails the whole block
		if storage.Pool.Transactions.Has(key) || reservoir.Has(txId) {
			return nil, nil, fault.ErrTransactionAlreadyExists
		}

		packed = append(packed, packedIssue...)
	}

	return issueStatus, packed, nil
}

// Bitmarks create
// --------------

type CreateArguments struct {
	Assets []transactionrecord.AssetData    `json:"assets"`
	Issues []transactionrecord.BitmarkIssue `json:"issues"`
}

type CreateReply struct {
	Assets     []AssetStatus    `json:"assets"`
	Issues     []IssueStatus    `json:"issues"`
	PayId      payment.PayId    `json:"payId"`
	PayNonce   payment.PayNonce `json:"payNonce"`
	Difficulty string           `json:"difficulty"`
	//PaymentAlternatives []block.MinerAddress `json:"paymentAlternatives"`// ***** FIX THIS: where to get addresses?

}

func (bitmarks *Bitmarks) Create(arguments *CreateArguments, reply *CreateReply) error {

	log := bitmarks.log
	assetCount := len(arguments.Assets)
	issueCount := len(arguments.Issues)

	if assetCount > maximumIssues || issueCount > maximumIssues {
		return fault.ErrTooManyItemsToProcess
	} else if 0 == assetCount && 0 == issueCount {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	log.Infof("Bitmarks.Create: %v", arguments)

	assetStatus, packedAssets, err := assetRegister(arguments.Assets)
	if nil != err {
		return err
	}

	issueStatus, packedIssues, err := bitmarksIssue(arguments.Issues)
	if nil != err {
		return err
	}

	result := CreateReply{
		Assets: assetStatus,
		Issues: issueStatus,
	}

	// fail if no data sent
	if 0 == len(packedAssets) || 0 == len(packedIssues) {
		return fault.ErrMissingParameters
	}
	// if data to send
	if 0 != len(packedAssets) {
		// announce transaction block to other peers
		messagebus.Bus.Broadcast.Send("assets", packedAssets)
	}

	if 0 != len(packedIssues) {

		// get here if all issues are new
		var d *difficulty.Difficulty
		newItem := false
		result.PayId, result.PayNonce, d, newItem = payment.Store(currency.Bitcoin, packedIssues, issueCount, true)
		result.Difficulty = d.GoString()

		// announce transaction block to other peers
		if newItem {
			messagebus.Bus.Broadcast.Send("issues", packedIssues)
		}
	}

	*reply = result
	return nil
	return nil
}

// Bitmarks proof
// --------------

type ProofArguments struct {
	PayId payment.PayId `json:"payId"`
	Nonce string        `json:"nonce"`
}

type ProofReply struct {
	Verified bool `json:"verified"`
}

func (bitmarks *Bitmarks) Proof(arguments *ProofArguments, reply *ProofReply) error {

	log := bitmarks.log

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	// arbitrary byte size limit
	size := hex.DecodedLen(len(arguments.Nonce))
	if size < 1 || size > payment.NonceLength {
		return fault.ErrInvalidNonce
	}

	log.Infof("proof for pay id: %v", arguments.PayId)
	log.Infof("client nonce: %q", arguments.Nonce)

	nonce := make([]byte, size)
	byteCount, err := hex.Decode(nonce, []byte(arguments.Nonce))
	if nil != err {
		return err
	}
	if byteCount != size {
		return fault.ErrInvalidNonce
	}

	log.Infof("client nonce hex: %x", nonce)

	// announce proof block to other peers
	packed := make([]byte, len(arguments.PayId), len(arguments.PayId)+len(nonce))
	copy(packed, arguments.PayId[:])
	packed = append(packed, nonce...)

	log.Infof("broadcast proof: %x", packed)
	messagebus.Bus.Broadcast.Send("proof", packed)

	// check if proof matches
	reply.Verified = payment.TryProof(arguments.PayId, nonce)

	return nil
}

// Bitmarks pay
// --------------

type PayArguments struct {
	PayId payment.PayId `json:"payId"` // id from the issue/transfer request
	// ***** FIX THIS: is currency required?
	//Currency currency.Currency `json:"currency"` // utf-8 → Enum
	Receipt string `json:"receipt"` // hex id from payment process
}

type PayReply struct {
	//Verified bool `json:"verified"`
}

func (bitmarks *Bitmarks) Pay(arguments *PayArguments, reply *PayReply) error {

	log := bitmarks.log

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	// arbitrary byte size limit
	size := hex.DecodedLen(len(arguments.Receipt))
	if size < 1 || size > payment.ReceiptLength {
		return fault.ErrReceiptTooLong
	}

	log.Infof("pay for pay id: %v", arguments.PayId)
	//log.Infof("currency: %q", arguments.Currency)
	log.Infof("receipt: %q", arguments.Receipt)

	// announce pay block to other peers
	packed := make([]byte, len(arguments.PayId))
	copy(packed, arguments.PayId[:])
	packed = append(packed, arguments.Receipt...)

	log.Infof("broadcast pay: %x", packed)
	messagebus.Bus.Broadcast.Send("pay", packed)

	payment.TrackPayment(arguments.PayId, arguments.Receipt, payment.RequiredConfirmations)

	return nil
}
