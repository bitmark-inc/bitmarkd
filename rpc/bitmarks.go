// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
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
	TxId      transactionrecord.Link `json:"txid"`
	Duplicate bool                   `json:"duplicate"`
}

type BitmarksIssueReply struct {
	Tx         []IssueStatus    `json:"tx"`
	PayId      payment.PayId    `json:"pay_id"`
	PayNonce   payment.PayNonce `json:"pay_nonce"`
	Difficulty string           `json:"difficulty"`
	//PaymentAddress []block.MinerAddress `json:"paymentAddress"`// ***** FIX THIS: where to get addresses?
	//Err       string `json:"error,omitempty"`
}

func (bitmarks *Bitmarks) Issue(arguments *[]transactionrecord.BitmarkIssue, reply *BitmarksIssueReply) error {

	log := bitmarks.log
	count := len(*arguments)

	if count > maximumIssues {
		return fault.ErrTooManyItemsToProcess
	}

	log.Infof("Bitmark.Issue: %v", arguments)

	result := BitmarksIssueReply{
		Tx: make([]IssueStatus, count),
	}

	//exists := true		// ***** FIX THIS: true only if all tx exist

	// pack each transaction
	packed := []byte{}
	for i, argument := range *arguments {

		packedIssue, err := argument.Pack(argument.Owner)
		if nil != err {
			return err
		}

		// ***** FIX THIS: to get the id
		// check record
		// id, oneExists := packedIssue.Exists()
		// log.Infof("Bitmark.Issue exists: %v", oneExists)
		// exists &= oneExists
		result.Tx[i].TxId = packedIssue.MakeLink() // ***** FIX THIS: replace with Exists() when code done

		log.Infof("packed issue: %x", packedIssue) // ***** FIX THIS: debugging
		log.Infof("id: %v", result.Tx[i].TxId)     // ***** FIX THIS: debugging

		packed = append(packed, packedIssue...)
	}

	var d *difficulty.Difficulty
	result.PayId, result.PayNonce, d = payment.Store(currency.Bitcoin, packed, count, true)
	result.Difficulty = d.GoString()
	// ***** FIX THIS: restore broadcasting
	// announce transaction block to system
	// if !exists {
	// 	messagebus.Send("", packed)
	// }

	*reply = result
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
