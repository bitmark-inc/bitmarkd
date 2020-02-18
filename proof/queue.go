// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
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
	"github.com/bitmark-inc/logger"
)

// PublishedItem - to send to proofer
type PublishedItem struct {
	Job    string             `json:"job"`
	Header blockrecord.Header `json:"header"`
	TxZero []byte             `json:"txZero"`
	TxIds  []merkle.Digest    `json:"txIds"`
}

// SubmittedItem - received from the proofer
type SubmittedItem struct {
	Request string `json:"request"`
	Job     string `json:"job"`
	Packed  []byte `json:"packed"`
	//***** FIX THIS: add for miner generated TxZero  []byte `json:"txZero"`
}

type entryType struct {
	item         *PublishedItem
	transactions []byte
}

// size of queue
const (
	queueSize = 32
)

// the queue
type jobQueueType struct {
	sync.RWMutex // to allow locking

	entries [queueSize]*entryType
	count   uint16
	clear   bool
}

// the queue storage
var jobQueue jobQueueType

// add job to the queue
func initialiseJobQueue() {
	jobQueue.Lock()
	defer jobQueue.Unlock()
	for i := range jobQueue.entries {
		jobQueue.entries[i] = nil
	}
	jobQueue.clear = false
}

// create a job number
func enqueueToJobQueue(item *PublishedItem, txdata []byte) {
	jobQueue.Lock()
	jobQueue.count += 1 // wraps (uint16)
	item.Job = fmt.Sprintf("%04x", jobQueue.count)
	n := jobQueue.count % queueSize
	if nil != jobQueue.entries[n] {
		jobQueue.entries[n].transactions = nil
		jobQueue.entries[n] = nil
	}
	jobQueue.entries[n] = &entryType{
		item:         item,
		transactions: txdata,
	}
	jobQueue.Unlock()
}

func matchToJobQueue(received *SubmittedItem, log *logger.L) (success bool) {
	jobQueue.Lock()
	defer jobQueue.Unlock()

	job := received.Job

	var entry *entryType
search:
	for _, e := range jobQueue.entries {
		if nil != e && nil != e.item && e.item.Job == job {
			entry = e
			break search
		}
	}

	if nil == entry {
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

		diff := entry.item.Header.Difficulty
		log.Debugf("incoming block difficulty: %f", diff.Value())

		if !digest.IsValidByDifficulty(diff, mode.ChainName()) {
			log.Infof("digest %s, difficulty %064x, digest not match difficulty criteria", digest.String(), diff.BigInt())
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
	for i := range jobQueue.entries {
		if nil != jobQueue.entries[i] {
			jobQueue.entries[i].transactions = nil
			jobQueue.entries[i] = nil
		}
	}
	jobQueue.clear = true

	return
}
