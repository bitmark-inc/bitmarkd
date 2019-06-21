// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"time"

	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/logger"
)

// Transaction - an RPC entry for transaction related functions
type Transaction struct {
	log     *logger.L
	limiter *rate.Limiter
	start   time.Time
}

// TransactionArguments - arguments for status RPC request
type TransactionArguments struct {
	TxId merkle.Digest `json:"txId"`
}

// TransactionStatusReply - results from status RPC
type TransactionStatusReply struct {
	Status string `json:"status"`
}

// Status - query transaction status
func (t *Transaction) Status(arguments *TransactionArguments, reply *TransactionStatusReply) error {

	if err := rateLimit(t.limiter); nil != err {
		return err
	}

	reply.Status = reservoir.TransactionStatus(arguments.TxId).String()
	return nil
}
