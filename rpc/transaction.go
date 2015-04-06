// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/base64"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

// Transaction
// -----------

type Transaction struct {
	log *logger.L
}

type PayArguments struct {
	Count    int    `json:"count"` // expected Bitmark transactions in payment, 0 => disable check
	Currency string `json:"currency"`
	Payment  string `json:"payment"`
}

type PayReply struct {
}

func (t *Transaction) Pay(arguments *PayArguments, reply *PayReply) error {

	t.log.Debugf("Pay arguments: %v", arguments)

	paymentData, err := hex.DecodeString(arguments.Payment) // try hex first
	if err != nil {                                         // if that fails -> try Base64
		paymentData, err = base64.StdEncoding.DecodeString(arguments.Payment)
		if nil != err {
			return err
		}
	}

	return payment.Pay(arguments.Currency, paymentData, arguments.Count)
}

// fetch some transactions
// -----------------------

type TransactionGetArguments struct {
	TxIds []transaction.Link `json:"txids"`
}

type TransactionGetReply struct {
	Transactions []transaction.Decoded `json:"transactions"`
}

func (t *Transaction) Get(arguments *TransactionGetArguments, reply *TransactionGetReply) error {

	// restrict arguments size to reasonable value
	size := len(arguments.TxIds)
	if size > MaximumGetSize {
		size = MaximumGetSize
	}

	txIds := arguments.TxIds[:size]

	reply.Transactions = transaction.Decode(txIds)
	return nil
}

// fetch all pending transactions
// ------------------------------

type TransactionPendingArguments struct {
}

type TransactionPendingReply struct {
	Transactions []transaction.Decoded `json:"transactions"`
}

func (t *Transaction) Pending(arguments *TransactionPendingArguments, reply *TransactionPendingReply) error {

	reply.Transactions = transaction.FetchPending()
	return nil
}
