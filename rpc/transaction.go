// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/logger"
)

// Transaction is a rpc entry for transaction related functions
type Transaction struct {
	log   *logger.L
	start time.Time
}

// TransactionArguments is the arguments for statuc rpc request
type TransactionArguments struct {
	TxId merkle.Digest `json:"txId"`
}

// TransactionStatus is a struct for an rpc reply
type TransactionStatusReply struct {
	Status string `json:"status"`
}

// Status is an rpc api for query transaction status
func (t *Transaction) Status(arguments *TransactionArguments, reply *TransactionStatusReply) error {
	reply.Status = reservoir.TransactionStatus(arguments.TxId).String()
	return nil
}
