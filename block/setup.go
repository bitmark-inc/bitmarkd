// Copyright (c) 2014-2018 Bitmark Inc.
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
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
)

// globals for background proccess
type blockData struct {
	sync.RWMutex // to allow locking

	log *logger.L

	height          uint64             // this is the current block Height
	previousBlock   blockdigest.Digest // and its digest
	previousVersion uint16             // plus its version

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
	globalData.log = log
	log.Info("starting…")

	// check storage is initialised
	if nil == storage.Pool.Blocks {
		log.Critical("storage pool is not initialise")
		return fault.ErrNotInitialised
	}

	// initialise block height and initial previous block digest
	globalData.height = genesis.BlockNumber
	globalData.previousBlock = genesis.LiveGenesisDigest
	globalData.previousVersion = 1
	if mode.IsTesting() {
		globalData.previousBlock = genesis.TestGenesisDigest
	}

	log.Infof("block height: %d", globalData.height)
	log.Infof("previous block: %v", globalData.previousBlock)
	log.Infof("previous version: %d", globalData.previousVersion)

	// fill ring with default values
	if err := fillRingBuffer(log); nil != err {
		return err
	}

	// initialise background tasks
	if err := globalData.blk.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	log.Info("start background…")

	processes := background.Processes{
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

	globalData.background.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// must hold lock to call this
func fillRingBuffer(log *logger.L) error {

	// reset ring to default
	blockring.Clear(log)

	// detect if any blocks on file
	if last, ok := storage.Pool.Blocks.LastElement(); ok {

		// get highest block
		packedHeader := blockrecord.PackedHeader(last.Value[:blockrecord.TotalBlockSize])
		header, err := packedHeader.Unpack()
		if nil != err {
			log.Criticalf("failed to unpack block: %d from storage  error: %s", binary.BigEndian.Uint64(last.Key), err)
			return err
		}
		globalData.previousVersion = header.Version
		globalData.previousBlock = packedHeader.Digest()
		globalData.height = header.Number // highest block number in database

		log.Infof("highest block from storage: %d", globalData.height)

		// determine the start point for fetching last few blocks
		n := genesis.BlockNumber + 1 // first real block (genesis block is not in db)
		if globalData.height > blockring.Size+1 {
			n = globalData.height - blockring.Size + 1
		}
		if n <= genesis.BlockNumber { // check just in case above calculation is wrong
			log.Criticalf("value of n < 2: %d", n)
			return fault.ErrInitialisationFailed
		}

		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, n)
		c := storage.Pool.Blocks.NewFetchCursor()
		c.Seek(key)

		items, err := c.Fetch(blockring.Size)
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
				log.Criticalf("failed to unpack block: %d from storage  error: %s", binary.BigEndian.Uint64(last.Key), err)
				return err
			}
			log.Infof("ring[%d] from block: %d", i, header.Number)

			// consistency check
			if n != header.Number {
				log.Criticalf("number mismatch actual: %d  expected: %d", header.Number, n)
				return fault.ErrInitialisationFailed
			}
			n += 1

			blockring.Put(header.Number, digest, item.Value)

			log.Tracec(func() string {
				// + begin debugging
				//log.Infof("header: %#v", header)

				data := item.Value[blockrecord.TotalBlockSize:]
				txs := make([]interface{}, header.TransactionCount)
			loop:
				for i := 1; true; i += 1 {
					transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
					if nil != err {
						//log.Errorf("tx[%d]: error: %s", i, err)
						//return err
						return fmt.Sprintf("tx[%d]: error: %s", i, err)
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
					return fmt.Sprintf("JSON marshal error: %s", err)
				}
				//log.Infof("block: %s", jsonData)
				return fmt.Sprintf("block: %s", jsonData)
				// - end debugging
			})
		}
	}
	return nil
}
