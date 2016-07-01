// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"encoding/binary"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"sync"
)

// to send to proofer
type PublishedItem struct {
	Job      string
	Header   blockrecord.Header
	Base     []byte
	TxIds    []merkle.Digest
	AssetIds []transactionrecord.AssetIndex
}

// received from the proofer
type SubmittedItem struct {
	Request string
	Job     string
	Packed  []byte
}

type entryType struct {
	item         *PublishedItem
	transactions []byte
}

// the queue
type jobQueueType struct {
	sync.RWMutex // to allow locking
	entries      map[string]*entryType
	count        uint16
	clear        bool
}

// the queue storage
var jobQueue jobQueueType

// add job to the queue
func initialiseJobQueue() {
	jobQueue.Lock()
	defer jobQueue.Unlock()
	jobQueue.entries = make(map[string]*entryType)
	jobQueue.clear = false
}

// create a job number
func enqueueToJobQueue(item *PublishedItem, txdata []byte) {
	jobQueue.Lock()
	jobQueue.count += 1 // wraps (uint16)
	job := fmt.Sprintf("%04x", jobQueue.count)
	item.Job = job
	jobQueue.entries[job] = &entryType{
		item:         item,
		transactions: txdata,
	}
	jobQueue.Unlock()
}

func matchToJobQueue(received *SubmittedItem) bool {
	jobQueue.Lock()
	defer jobQueue.Unlock()

	job := received.Job

	entry, ok := jobQueue.entries[job]
	if !ok {
		return false
	}

	// get current difficulty
	difficulty := entry.item.Header.Difficulty.BigInt()

	switch received.Request {

	case "block.nonce":
		if len(received.Packed) != blockrecord.NonceSize {
			return false
		}
		entry.item.Header.Nonce = blockrecord.NonceType(binary.LittleEndian.Uint64(received.Packed))
		ph := entry.item.Header.Pack()
		digest := ph.Digest()
		if digest.Cmp(difficulty) > 0 {
			return false
		}
		packedBlock := ph //make([]byte,len(ph)+len(entry.item.Base)+len(entry.transactions))
		packedBlock = append(packedBlock, entry.item.Base...)
		packedBlock = append(packedBlock, entry.transactions...)

		// ***** FIX THIS: broadcast this packedBlock
		// ***** FIX THIS: ==========================

		blockNumber := make([]byte, 8)
		binary.BigEndian.PutUint64(blockNumber, entry.item.Header.Number)

		// store the entrire block
		storage.Pool.Blocks.Put(blockNumber, packedBlock)

		// update global block data
		block.Set(&entry.item.Header)

		for _, txId := range entry.item.TxIds {
			key := txId[:]
			data := storage.Pool.VerifiedTransactions.Get(key)
			if nil != data {
				storage.Pool.Transactions.Put(key, data)
				storage.Pool.VerifiedTransactions.Delete(key)
			}
		}
		for _, assetId := range entry.item.AssetIds {
			key := assetId[:]
			data := storage.Pool.VerifiedAssets.Get(key)
			if nil != data {
				storage.Pool.Assets.Put(key, data)
				storage.Pool.VerifiedAssets.Delete(key)
			}
		}
	}

	// erase the queue
	jobQueue.entries = make(map[string]*entryType)
	jobQueue.clear = true

	return true
}
