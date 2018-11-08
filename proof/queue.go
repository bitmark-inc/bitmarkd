// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
)

// to send to proofer
type PublishedItem struct {
	Job    string
	Header blockrecord.Header
	TxZero []byte
	TxIds  []merkle.Digest
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

func matchToJobQueue(received *SubmittedItem) (success bool) {
	jobQueue.Lock()
	defer jobQueue.Unlock()

	job := received.Job

	entry, ok := jobQueue.entries[job]
	if !ok {
		return
	}

	// if not normal abandon the queue and the submission
	if !mode.Is(mode.Normal) {
		goto cleanup
	}

	switch received.Request {

	case "block.nonce":
		if len(received.Packed) != blockrecord.NonceSize {
			return
		}
		entry.item.Header.Nonce = blockrecord.NonceType(binary.LittleEndian.Uint64(received.Packed))
		ph := entry.item.Header.Pack()
		digest := ph.Digest()

		// get current difficulty
		difficulty := entry.item.Header.Difficulty.BigInt()

		if digest.Cmp(difficulty) > 0 {
			return
		}
		packedBlock := ph[:] //make([]byte,len(ph)+len(entry.item.Base)+len(entry.transactions))
		packedBlock = append(packedBlock, entry.item.TxZero...)
		packedBlock = append(packedBlock, entry.transactions...)

		// broadcast this packedBlock for processing
		messagebus.Bus.Blockstore.Send("local", packedBlock)
		success = true
	}

cleanup:
	// erase the queue
	jobQueue.entries = make(map[string]*entryType)
	jobQueue.clear = true

	return
}
