// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/rpc/ratelimit"

	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/logger"
)

const (
	rateLimitTransaction = 200
	rateBurstTransaction = 100
)

// Transaction - an RPC entry for transaction related functions
type Transaction struct {
	Log     *logger.L
	Limiter *rate.Limiter
	Start   time.Time
	Rsvr    reservoir.Reservoir
}

// TransactionArguments - arguments for status RPC request
type TransactionArguments struct {
	TxId merkle.Digest `json:"txId"`
}

// TransactionStatusReply - results from status RPC
type TransactionStatusReply struct {
	Status string `json:"status"`
}

func New(log *logger.L, start time.Time, rsvr reservoir.Reservoir) Transaction {
	return Transaction{
		Log:     log,
		Limiter: rate.NewLimiter(rateLimitTransaction, rateBurstTransaction),
		Start:   start,
		Rsvr:    rsvr,
	}
}

// Status - query transaction status
func (t *Transaction) Status(arguments *TransactionArguments, reply *TransactionStatusReply) error {

	if err := ratelimit.Limit(t.Limiter); nil != err {
		return err
	}

	reply.Status = t.Rsvr.TransactionStatus(arguments.TxId).String()
	return nil
}
