// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
)

// internal constants
const (
	ringSize = 20 // size of ring buffer
)

// type to hold a block's digest and its crc64 check code
type ringBuffer struct {
	number uint64             // block number
	crc    uint64             // CRC64_ECMA(block_number, complete_block_bytes)
	digest blockdigest.Digest // header digest
}

// globals for background proccess
type blockData struct {
	sync.RWMutex // to allow locking

	log *logger.L

	height        uint64             // this is the current block Height
	previousBlock blockdigest.Digest // and its digest

	ring      [ringSize]ringBuffer
	ringIndex int

	blk blockstore // for sequencing block storage

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData blockData

// setup the current block data
func Initialise() error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	log := logger.New("block")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log = log
	log.Info("starting…")

	// check storage is initialised
	if nil == storage.Pool.Blocks {
		log.Critical("storage pool is not initialise")
		return fault.ErrNotInitialised
	}

	// initialise block height and fill ring with default values
	if err := clearRingBuffer(log); nil != err {
		return err
	}

	log.Infof("block height: %d", globalData.height)
	log.Infof("previous block: %v", globalData.previousBlock)
	for i := range globalData.ring {
		p := "  "
		if i == globalData.ringIndex {
			p = "->"
		}
		log.Infof("%sring[%02d]: number: %d crc: 0x%015x  digest: %v", p, i, globalData.ring[i].number, globalData.ring[i].crc, globalData.ring[i].digest)
	}

	// initialise background tasks
	if err := globalData.blk.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	log.Info("start background…")

	var processes = background.Processes{
		&globalData.blk,
	}

	globalData.background = background.Start(processes, log)

	return nil
}

// shudown the block system
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// finally...
	globalData.initialised = false

	return nil
}

// must hold lock to call this
func clearRingBuffer(log *logger.L) error {

	// set initial crc depending on mode
	globalData.height = genesis.BlockNumber
	globalData.previousBlock = genesis.LiveGenesisDigest
	block := genesis.LiveGenesisBlock
	if mode.IsTesting() {
		globalData.previousBlock = genesis.TestGenesisDigest
		block = genesis.TestGenesisBlock
	}

	// default CRC of appropeiate genesis block
	crc := CRC(globalData.height, block)

	// fill ring with default values
	globalData.ringIndex = 0
	for i := 0; i < len(globalData.ring); i += 1 {
		globalData.ring[i].number = globalData.height
		globalData.ring[i].digest = globalData.previousBlock
		globalData.ring[i].crc = crc
	}

	// detect if any blocks on file
	if last, ok := storage.Pool.Blocks.LastElement(); ok {

		// get highest block
		packedHeader := blockrecord.PackedHeader(last.Value[:blockrecord.TotalBlockSize])
		header, err := packedHeader.Unpack()
		if nil != err {
			log.Criticalf("failed to unpack block: %d from storage  error: %v", binary.BigEndian.Uint64(last.Key), err)
			return err
		}
		globalData.previousBlock = packedHeader.Digest()
		globalData.height = header.Number // highest block number in database

		log.Infof("highest block from storage: %d", globalData.height)

		// determine the start point for fetching last few blocks
		n := genesis.BlockNumber + 1 // first real block (genesis block is not in db)
		if globalData.height > ringSize+1 {
			n = globalData.height - ringSize + 1
		}
		if n <= genesis.BlockNumber { // check just in case above calculation is wrong
			log.Criticalf("value of n < 2: %d", n)
			return fault.ErrInitialisationFailed
		}

		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, n)
		c := storage.Pool.Blocks.NewFetchCursor()
		c.Seek(key)

		items, err := c.Fetch(len(globalData.ring))
		if nil != err {
			return err
		}

		log.Infof("populate ring buffer staring from block: %d", n)
		log.Tracef("items: %x", items)

		for i, item := range items {

			packedHeader := blockrecord.PackedHeader(item.Value[:blockrecord.TotalBlockSize])
			digest := packedHeader.Digest()
			header, err := packedHeader.Unpack()
			if nil != err {
				log.Criticalf("failed to unpack block: %d from storage  error: %v", binary.BigEndian.Uint64(last.Key), err)
				return err
			}
			log.Infof("ring[%d] from block: %d", i, header.Number)

			// consistency check
			if n != header.Number {
				log.Criticalf("number mismatch actual: %d  expected: %d", header.Number, n)
				return fault.ErrInitialisationFailed
			}
			n += 1

			globalData.ring[i].number = header.Number
			globalData.ring[i].digest = digest
			globalData.ring[i].crc = CRC(header.Number, item.Value)

			log.Tracec(func() string {
				// + begin debugging
				//log.Infof("header: %#v", header)

				data := item.Value[blockrecord.TotalBlockSize:]
				txs := make([]interface{}, header.TransactionCount)
			loop:
				for i := 1; true; i += 1 {
					transaction, n, err := transactionrecord.Packed(data).Unpack()
					if nil != err {
						//log.Errorf("tx[%d]: error: %v", i, err)
						//return err
						return fmt.Sprintf("tx[%d]: error: %v", i, err)
					}
					txs[i-1] = transaction
					data = data[n:]
					if 0 == len(data) {
						break loop
					}
				}
				s := struct {
					Header       *blockrecord.Header
					Transactions []interface{}
				}{
					Header:       header,
					Transactions: txs,
				}
				jsonData, err := json.MarshalIndent(s, "", "  ")
				if nil != err {
					//return err
					return fmt.Sprintf("JSON marshal error: %v", err)
				}
				//log.Infof("block: %s", jsonData)
				return fmt.Sprintf("block: %s", jsonData)
				// - end debugging
			})
		}
		globalData.ringIndex += len(items)
		if globalData.ringIndex >= len(globalData.ring) {
			globalData.ringIndex = 0
		}
	}
	return nil
}
