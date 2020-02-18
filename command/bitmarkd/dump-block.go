// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

type transactionItem struct {
	Index int           `json:"index"`
	TxId  merkle.Digest `json:"txId"`
	Type  string        `json:"type"`
	Data  interface{}   `json:"data"`
}

type blockResult struct {
	Digest       blockdigest.Digest  `json:"digest"`
	Header       *blockrecord.Header `json:"header"`
	Transactions []transactionItem   `json:"transactions"`
}

// dump of a particular block
func dumpBlock(number uint64) (*blockResult, error) {

	// fetch block and compute digest
	n := make([]byte, 8)
	binary.BigEndian.PutUint64(n, number)

	packed := storage.Pool.Blocks.Get(n)
	if nil == packed {
		return nil, fault.BlockNotFound
	}

	header, digest, data, err := blockrecord.ExtractHeader(packed, number, false)
	if nil != err {
		return nil, err
	}

	txs := make([]transactionItem, header.TransactionCount)
loop:
	for i := 1; true; i += 1 {
		transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
		if nil != err {
			return nil, err
		}
		name, _ := transactionrecord.RecordName(transaction)
		txs[i-1] = transactionItem{
			Index: i,
			TxId:  merkle.NewDigest(data[:n]),
			Type:  name,
			Data:  transaction,
		}
		data = data[n:]
		if 0 == len(data) {
			break loop
		}
	}

	result := &blockResult{
		Digest:       digest,
		Header:       header,
		Transactions: txs,
	}

	return result, nil
}
