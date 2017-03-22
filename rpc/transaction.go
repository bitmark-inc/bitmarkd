// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/logger"
	"time"
)

// Transaction is a rpc entry for transaction related functions
type Transaction struct {
	log   *logger.L
	start time.Time
}

// TransactionArguments is the arguments for statuc rpc request
type TransactionArguments struct {
	Id string `json:"txId"`
}

// TransactionStatus is a struct for an rpc reply
type TransactionStatus struct {
	Status string `json:"status"`
}

// Status is an rpc api for query transaction status
func (t *Transaction) Status(arguments *TransactionArguments, reply *TransactionStatus) error {
	var txId merkle.Digest
	err := txId.UnmarshalText([]byte(arguments.Id))
	if err != nil {
		return err
	}
	reply.Status = reservoir.TransactionStatus(txId).String()
	return nil
}
