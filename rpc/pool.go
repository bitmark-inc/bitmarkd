// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

// --------------------

// e.g.
// {"id":1,"method":"Pool.List","params":[{"Index":0,"Count":10}]}

type Pool struct {
	log *logger.L
}

type PoolArguments struct {
	Index *transaction.IndexCursor `json:"index"`
	Count int                      `json:"count"`
}

type PoolReply struct {
	Transactions []transaction.PoolResult `json:"transactions"`
	NextIndex    transaction.IndexCursor  `json:"nextIndex"`
}

func (pool *Pool) List(arguments *PoolArguments, reply *PoolReply) error {
	if arguments.Count <= 0 {
		arguments.Count = 10
	}
	index := arguments.Index
	if nil == index {
		c := transaction.IndexCursor(0)
		index = &c
	}
	txs := index.FetchPool(arguments.Count)
	for _, e := range txs {
		reply.Transactions = append(reply.Transactions, e)
	}
	reply.NextIndex = *index
	return nil
}
