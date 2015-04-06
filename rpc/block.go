// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
	"time"
)

// Block
// -------

type Block struct {
	log *logger.L
}

// current block number
// --------------------

// number arguments
type NumberArguments struct {
}

// number reply
type NumberReply struct {
	Number uint64 `json:"number"`
}

// Block.Number function
func (blk *Block) Number(arguments *NumberArguments, reply *NumberReply) error {

	// return the highest block number that is currently stored
	reply.Number = block.Number() - 1

	return nil
}

// Block get
// ---------

type BlockGetArguments struct {
	Number uint64 `json:"number"`
}

type BlockGetReply struct {
	Digest       block.Digest          `json:"digest"`
	Number       uint64                `json:"number"`
	Timestamp    time.Time             `json:"timestamp"`
	Transactions []transaction.Decoded `json:"transactions"`
}

func (blk *Block) Get(arguments *BlockGetArguments, reply *BlockGetReply) error {
	log := blk.log

	log.Infof("Block.get: %v", arguments)

	packed, found := block.Get(arguments.Number)
	if !found {
		return fault.ErrBlockNotFound
	}

	var theBlock block.Block
	err := packed.Unpack(&theBlock)
	if nil != err {
		return err
	}

	reply.Digest = theBlock.Digest
	reply.Number = theBlock.Number
	reply.Timestamp = theBlock.Timestamp
	// reply.RawAddress = theBlock.RawAddress

	size := len(theBlock.TxIds)
	if 0 == size {
		return nil
	}

	// kind of awkward since Link and Digest are the same type
	// but cannot cats arrays directly
	txIds := make([]transaction.Link, len(theBlock.TxIds))
	for i, v := range theBlock.TxIds {
		txIds[i] = transaction.Link(v)
	}

	reply.Transactions = transaction.Decode(txIds)

	return nil
}
