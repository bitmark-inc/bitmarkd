// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/merkle"
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

	globalData.log = logger.New("block")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	// set initial data
	globalData.height = genesis.BlockNumber
	globalData.previousBlock = genesis.LiveGenesisDigest
	block := genesis.LiveGenesisBlock
	if mode.IsTesting() {
		globalData.previousBlock = genesis.TestGenesisDigest
		block = genesis.TestGenesisBlock
	}

	// check storage is initialised
	if nil == storage.Pool.Blocks {
		globalData.log.Critical("storage pool is not initialise")
		return fault.ErrNotInitialised
	}

	// fill ring with default values
	globalData.ringIndex = 0
	crc := CRC(globalData.height, block)
	for i := 0; i < len(globalData.ring); i += 1 {
		globalData.ring[i].number = globalData.height
		globalData.ring[i].digest = globalData.previousBlock
		globalData.ring[i].crc = crc
	}

	if last, ok := storage.Pool.Blocks.LastElement(); ok {
		packedHeader := blockrecord.PackedHeader(last.Value[:blockrecord.TotalBlockSize])
		header, err := packedHeader.Unpack()
		if nil != err {
			globalData.log.Criticalf("failed to unpack block: %d from storage  error: %v", binary.BigEndian.Uint64(last.Key), err)
			return err
		}
		globalData.previousBlock = packedHeader.Digest()
		globalData.height = header.Number // highest block number in database

		// determine the start point for fetching last few blocks
		n := genesis.BlockNumber + 1 // first real block (genesis block is not in db)
		if globalData.height > ringSize+1 {
			n = globalData.height - ringSize
		}
		if n <= genesis.BlockNumber { // check just in case above calculation is wrong
			globalData.log.Criticalf("value of n < 2: %d", n)
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

		for i, item := range items {
			packedHeader := blockrecord.PackedHeader(item.Value[:blockrecord.TotalBlockSize])
			digest := packedHeader.Digest()
			header, err := packedHeader.Unpack()
			if nil != err {
				globalData.log.Criticalf("failed to unpack block: %d from storage  error: %v", binary.BigEndian.Uint64(last.Key), err)
				return err
			}
			// consistency checkblock.ringBuffer{crc:0x82ea2dc4e90280ae
			if n != header.Number {
				globalData.log.Criticalf("number mismatch actual: %d  expected: %d", header.Number, n)
				return fault.ErrInitialisationFailed
			}
			n += 1

			globalData.ring[i].number = header.Number
			globalData.ring[i].digest = digest
			globalData.ring[i].crc = CRC(header.Number, item.Value)

			// ***** FIX THIS: debugging
			//globalData.log.Infof("header: %#v", header)

			data := item.Value[blockrecord.TotalBlockSize:]
			txs := make([]interface{}, header.TransactionCount)
		loop:
			for i := 1; true; i += 1 { // ***** FIX THIS: debugging
				transaction, n, err := transactionrecord.Packed(data).Unpack()
				if nil != err {
					globalData.log.Errorf("tx[%d]: error: %v", i, err)
					return err
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
				return err
			}
			globalData.log.Infof("block: %s", jsonData) // ***** FIX THIS: debugging
			// ***** FIX THIS: end debugging

		}
		globalData.ringIndex += len(items)
		if globalData.ringIndex >= len(globalData.ring) {
			globalData.ringIndex = 0
		}
	}

	globalData.log.Infof("block height: %d", globalData.height)
	globalData.log.Infof("previous block: %v", globalData.previousBlock)
	for i := range globalData.ring {
		p := "  "
		if i == globalData.ringIndex {
			p = "->"
		}
		globalData.log.Infof("%sring[%02d]: number: %d crc: 0x%015x  digest: %v", p, i, globalData.ring[i].number, globalData.ring[i].crc, globalData.ring[i].digest)
	}

	// all data initialised
	globalData.initialised = true

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

// get block data for initialising a new block
// returns: previous block digest and the number for the new block
func Get() (blockdigest.Digest, uint64) {
	globalData.Lock()
	defer globalData.Unlock()
	nextBlockNumber := globalData.height + 1
	return globalData.previousBlock, nextBlockNumber
}

// get the current height
func GetHeight() uint64 {
	globalData.Lock()
	height := globalData.height
	globalData.Unlock()
	return height
}

func DigestForBlock(number uint64) (blockdigest.Digest, error) {
	globalData.Lock()
	defer globalData.Unlock()

	// valid block number
	if number <= genesis.BlockNumber {
		if mode.IsTesting() {
			return genesis.TestGenesisDigest, nil
		}
		return genesis.LiveGenesisDigest, nil
	}

	// check if in the cache
	if number > genesis.BlockNumber && number <= globalData.height {
		i := globalData.height - number
		if i < ringSize {
			j := globalData.ringIndex - 1 - int(i)
			if j < 0 {
				j = ringSize - 1
			}
			if number != globalData.ring[j].number {
				fault.Panicf("block.DigestForBlock: ring buffer corrupted block number, actual: %d  expected: %d", globalData.ring[j].number, number)
			}
			return globalData.ring[j].digest, nil
		}
	}

	// no cache, fetch block and compute digest
	n := make([]byte, 8)
	binary.BigEndian.PutUint64(n, number)
	packed := storage.Pool.Blocks.Get(n) // ***** FIX THIS: possible optimisation is to store the block hashes in a separate index
	if nil == packed {
		return blockdigest.Digest{}, fault.ErrBlockNotFound
	}
	return blockrecord.PackedHeader(packed).Digest(), nil
}

// store the block and update block data
func Store(header *blockrecord.Header, digest blockdigest.Digest, packedBlock []byte) {
	globalData.Lock()
	//defer globalData.Unlock()

	expectedBlockNumber := globalData.height + 1
	if expectedBlockNumber != header.Number {
		fault.Panicf("block.Store: out of sequence block: actual: %d  expected: %d", header.Number, expectedBlockNumber)
	}

	globalData.previousBlock = digest
	globalData.height = header.Number

	i := globalData.ringIndex
	globalData.ring[i].number = header.Number
	globalData.ring[i].digest = digest
	globalData.ring[i].crc = CRC(header.Number, packedBlock)
	i = i + 1
	if i >= len(globalData.ring) {
		i = 0
	}
	globalData.ringIndex = i

	// end of critical section
	globalData.Unlock()

	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	storage.Pool.Blocks.Put(blockNumber, packedBlock)
}

// store an incoming block checking to make sure it is valid first
func StoreIncoming(packedBlock []byte) error {
	packedHeader := blockrecord.PackedHeader(packedBlock[:blockrecord.TotalBlockSize])
	header, err := packedHeader.Unpack()
	if nil != err {
		return err
	}

	if globalData.previousBlock != header.PreviousBlock {
		return fault.ErrPreviousBlockDigestDoesNotMatch
	}

	data := packedBlock[blockrecord.TotalBlockSize:]

	type txn struct {
		packed   transactionrecord.Packed
		unpacked interface{}
	}

	txs := make([]txn, header.TransactionCount)
	txIds := make([]merkle.Digest, header.TransactionCount)

	// check all transactions are valid
	for i := uint16(0); i < header.TransactionCount; i += 1 {
		transaction, n, err := transactionrecord.Packed(data).Unpack()
		if nil != err {
			return err
		}

		txs[i].packed = transactionrecord.Packed(data[:n])
		txs[i].unpacked = transaction
		txIds[i] = merkle.NewDigest(data[:n])
		data = data[n:]
	}

	// build the tree of transaction IDs
	fullMerkleTree := merkle.FullMerkleTree(txIds)
	merkleRoot := fullMerkleTree[len(fullMerkleTree)-1]

	if merkleRoot != header.MerkleRoot {
		return fault.ErrMerkleRootDoesNotMatch
	}

	digest := packedHeader.Digest()
	Store(header, digest, packedBlock)

	// store transactions
	for i, txn := range txs {
		txid := txIds[i]
		packed := txn.packed
		transaction := txn.unpacked
		switch transaction.(type) {

		case *transactionrecord.AssetData:
			asset := transaction.(*transactionrecord.AssetData)
			assetIndex := asset.AssetIndex()
			key := assetIndex[:]
			storage.Pool.VerifiedAssets.Delete(key)
			storage.Pool.Assets.Put(key, packed)

		case *transactionrecord.BaseData:
			// currently not stored separately

		default: // just store the transaction
			key := txid[:]
			storage.Pool.VerifiedTransactions.Delete(key)
			storage.Pool.Transactions.Put(key, packed)
		}
	}

	return nil
}

// fetch latest crc value
func GetLatestCRC() uint64 {
	globalData.Lock()
	i := globalData.ringIndex - 1
	if i < 0 {
		i = len(globalData.ring) - 1
	}
	crc := globalData.ring[i].crc
	globalData.Unlock()
	return crc
}
