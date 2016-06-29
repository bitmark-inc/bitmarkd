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
	"github.com/bitmark-inc/bitmarkd/payment"
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
	TxId      transactionrecord.Link `json:"txId"`
	Duplicate bool                   `json:"duplicate"` // ***** FIX THIS: is this necessary?
}

type BitmarksIssueReply struct {
	Transactions []IssueStatus    `json:"transactions"`
	PayId        payment.PayId    `json:"payId"`
	PayNonce     payment.PayNonce `json:"payNonce"`
	Difficulty   string           `json:"difficulty"`
	//PaymentAlternatives []block.MinerAddress `json:"paymentAlternatives"`// ***** FIX THIS: where to get addresses?
}

func (bitmarks *Bitmarks) Issue(arguments *[]transactionrecord.BitmarkIssue, reply *BitmarksIssueReply) error {

	log := bitmarks.log
	count := len(*arguments)

	if count > maximumIssues {
		return fault.ErrTooManyItemsToProcess
	}

	log.Infof("Bitmarks.Issue: %v", arguments)

	result := BitmarksIssueReply{
		Transactions: make([]IssueStatus, count),
	}

	//exists := true		// ***** FIX THIS: true only if all transactions exist

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

		// ***** FIX THIS: should exists only consider verified/confirmed
		// ***** FIX THIS: then abort if even one transaction "exists" since it has already been paid?
		// ***** FIX THIS: to get the id
		// check record
		// id, oneExists := packedIssue.Exists()
		// log.Infof("Bitmark.Issue exists: %v", oneExists)
		// exists &= oneExists
		result.Transactions[i].TxId = packedIssue.MakeLink() // ***** FIX THIS: replace with Exists() when code done

		log.Infof("packed issue: %x", packedIssue)       // ***** FIX THIS: debugging
		log.Infof("id: %v", result.Transactions[i].TxId) // ***** FIX THIS: debugging

		packed = append(packed, packedIssue...)
	}

	var d *difficulty.Difficulty
	result.PayId, result.PayNonce, d = payment.Store(currency.Bitcoin, packed, count, true) // ***** FIX THIS: need actual currency value, not constant
	result.Difficulty = d.GoString()

	// ***** FIX THIS: restore broadcasting
	// announce transaction block to other peers
	// if !exists {
	// 	messagebus.Send("", packed)
	// }

	*reply = result
	return nil
}

// Bitmarks proof
// --------------

type Proofarguments struct {
	PayId payment.PayId `json:"payId"`
	Nonce string        `json:"nonce"`
}

type ProofReply struct {
	Verified bool `json:"verified"`
}

func (bitmarks *Bitmarks) Proof(arguments *Proofarguments, reply *ProofReply) error {

	log := bitmarks.log

	log.Infof("proof for pay id: %x", arguments.PayId)
	log.Infof("client nonce: %q", arguments.Nonce)

	size := hex.DecodedLen(len(arguments.Nonce))

	// arbitrary byte size limit
	if size < 1 || size > 16 {
		return fault.ErrInvalidNonce
	}

	nonce := make([]byte, size)
	byteCount, err := hex.Decode(nonce, []byte(arguments.Nonce))
	if nil != err {
		return err
	}
	if byteCount != size {
		return fault.ErrInvalidNonce
	}

	log.Infof("client nonce hex: %x", nonce)

	reply.Verified = payment.TryProof(arguments.PayId, nonce)
	if reply.Verified {
		log.Warn("***need to broadcast pay id pay nonce and client nonce to peers") // ***** FIX THIS: add real broadcast
	}

	return nil
}

// // Bitmarks transfer
// // -----------------

// func (bitmarks *Bitmarks) Transfer(arguments *[]transaction.BitmarkTransfer, reply *[]BitmarkTransferReply) error {

// 	bitmark := bitmarks.bitmark

// 	result := make([]BitmarkTransferReply, len(*arguments))
// 	for i, argument := range *arguments {
// 		if err := bitmark.Transfer(&argument, &result[i]); err != nil {
// 			result[i].Err = err.Error()
// 		}
// 	}

// 	*reply = result
// 	return nil
// }
