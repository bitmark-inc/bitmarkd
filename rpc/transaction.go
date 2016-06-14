// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

// import (
// 	"encoding/hex"
// 	"github.com/bitmark-inc/bitmarkd/transactionrecord"
// 	"github.com/bitmark-inc/logger"
// )

// // Transaction
// // -----------

// type Transaction struct {
// 	log *logger.L
// }

// // fetch some transactions
// // -----------------------

// type TransactionGetArguments struct {
// 	TxIds []transaction.Link `json:"txids"`
// }

// type TransactionGetReply struct {
// 	Transactions []transaction.Decoded `json:"transactions"`
// }

// func (t *Transaction) Get(arguments *TransactionGetArguments, reply *TransactionGetReply) error {

// 	// restrict arguments size to reasonable value
// 	size := len(arguments.TxIds)
// 	if size > MaximumGetSize {
// 		size = MaximumGetSize
// 	}

// 	txIds := arguments.TxIds[:size]

// 	reply.Transactions = transaction.Decode(txIds)
// 	return nil
// }

// // fetch all pending transactions
// // ------------------------------

// type TransactionPendingArguments struct {
// }

// type TransactionPendingReply struct {
// 	Transactions []transaction.Decoded `json:"transactions"`
// }

// func (t *Transaction) Pending(arguments *TransactionPendingArguments, reply *TransactionPendingReply) error {

// 	reply.Transactions = transaction.FetchPending()
// 	return nil
// }
