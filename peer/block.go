// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
)

type Block struct {
	log *logger.L
}

// ------------------------------------------------------------

type BlockNumberArguments struct {
}

type BlockNumberReply struct {
	Number uint64
}

// fetch the highest block number available
func (t *Block) Number(arguments *BlockNumberArguments, reply *BlockNumberReply) error {
	// fetch the next block number to be mined and adjust(-1) to
	// highest block actually available
	reply.Number = block.Number() - 1
	return nil
}

// ------------------------------------------------------------

type BlockPutArguments struct {
	Bilateral_SENDER string // magick field
	Block            block.Packed
}

type BlockPutReply struct {
	Duplicate bool
}

// new incoming block
func (t *Block) Put(arguments *BlockPutArguments, reply *BlockPutReply) error {

	t.log.Infof("received block: %x...", arguments.Block[:32]) // only show first 32 bytes

	packedBlock := block.Packed(arguments.Block)

	var blk block.Block
	err := packedBlock.Unpack(&blk)
	if nil != err {
		t.log.Errorf("received block: error: %v", err)
		return err
	}

	// propagate
	if blk.Number >= block.Number() {
		t.log.Infof("propagate: block: %d", blk.Number)
		blockPair := BlockPair{
			unpacked: blk,
			packed:   packedBlock,
		}
		from := arguments.Bilateral_SENDER
		messagebus.Send(from, blockPair)
	}

	return nil
}

// ------------------------------------------------------------

type BlockGetArguments struct {
	Number uint64
}

type BlockGetReply struct {
	Data []byte
}

// read a specific block
func (t *Block) Get(arguments *BlockGetArguments, reply *BlockGetReply) error {
	data, found := block.Get(arguments.Number)
	if !found {
		return fault.ErrBlockNotFound
	}
	reply.Data = data
	return nil
}
