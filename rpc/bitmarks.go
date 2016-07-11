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
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/payment"
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

type BitmarksIssueReply struct {
	TxIds      []transactionrecord.Link `json:"txIds"`
	PayId      payment.PayId            `json:"payId"`
	PayNonce   payment.PayNonce         `json:"payNonce"`
	Difficulty string                   `json:"difficulty"`
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

	result := BitmarksIssueReply{
		TxIds: make([]transactionrecord.Link, count),
	}

	// pack each transaction
	packed := []byte{}
	for i, argument := range *arguments {

		packedIssue, err := argument.Pack(argument.Owner)
		if nil != err {
			return err
		}

		if !asset.Exists(argument.AssetIndex) {
			return fault.ErrAssetNotFound
		}

		link := packedIssue.MakeLink()
		result.TxIds[i] = link
		key := link[:]

		// even a single verified/confirmed issue fails the whole block
		if storage.Pool.Transactions.Has(key) || storage.Pool.VerifiedTransactions.Has(key) {
			return fault.ErrTransactionAlreadyExists
		}

		log.Tracef("packed issue[%d]: %x", i, packedIssue)
		log.Debugf("id[%d]: %v", i, link)

		packed = append(packed, packedIssue...)
	}

	// fail if no data sent
	if 0 == len(packed) {
		return fault.ErrMissingParameters
	}

	// get here if all issues are new
	var d *difficulty.Difficulty
	result.PayId, result.PayNonce, d = payment.Store(currency.Bitcoin, packed, count, true)
	result.Difficulty = d.GoString()

	// announce transaction block to other peers
	messagebus.Send("issues", packed)

	*reply = result
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
	if size < 1 || size > 16 {
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
	messagebus.Send("proof", packed)

	// check if proof matches
	reply.Verified = payment.TryProof(arguments.PayId, nonce)

	return nil
}
