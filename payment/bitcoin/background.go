// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	// "bytes"
	"container/heap"
	// "encoding/binary"
	// "github.com/bitmark-inc/bitmarkd/block"
	//"github.com/bitmark-inc/bitmarkd/account"
	//"github.com/bitmark-inc/bitmarkd/currency"
	// "github.com/bitmark-inc/bitmarkd/counter"
	// "github.com/bitmark-inc/bitmarkd/difficulty"
	// "github.com/bitmark-inc/bitmarkd/fault"
	// "github.com/bitmark-inc/bitmarkd/pool"
	//"github.com/bitmark-inc/bitmarkd/transactionrecord"
	//"github.com/bitmark-inc/logger"
	//"sync"
	// "time"
)

// accept a new payment to monitor
func QueueItem(payId PayId, txId string, confirmations uint64) {
	globalData.itemQueue <- &priorityItem{
		payId:         payId,
		txId:          txId,
		confirmations: confirmations,
		blockNumber:   globalData.latestBlockNumber + confirmations,
	}
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (state *bitcoinData) Run(args interface{}, shutdown <-chan struct{}) {

	log := state.log

	pq := new(priorityQueue)
	heap.Init(pq)

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case blockNumber := <-state.blockQueue:
			//state.latestBlockNumber := blockNumber
			process(pq, blockNumber)
		case item := <-state.itemQueue:
			//item.blockNumber = state.latestBlockNumber + item.confirmations
			heap.Push(pq, item)
		}
	}
}

// process all items <= block number
func process(pq *priorityQueue, blockNumber uint64) {

loop:
	for pq.Len() > 0 {
		item := heap.Pop(pq).(*priorityItem)
		//log.Infof("check payId: %x ", item.payId)

		// if cannot possibly reach required confirmations
		if item.blockNumber > blockNumber {
			heap.Push(pq, item)
			break loop
		}

		// could reach confirmations
		var reply bitcoinTransaction

		// fetch transaction and decode
		err := bitcoinGetRawTransaction(item.txId, &reply)
		if nil != err {
			////////now what // ***** FIX THIS:
		}

		ok := false
		for _, vout := range reply.Vout {
			if OP_RETURN == vout.ScriptPubKey.Hex[0:2] {
				//***** FIX THIS: && vout.ScriptPubKey.Hex[2:] == item.payId { // need hexstring→[]byte
				ok = true
				break
			}
		}
		if !ok {
			// log.Errorf("no payId: %x in tx: %x ", item.payId, item.txId)
			////////now what // ***** FIX THIS:
		}

		if reply.Confirmations >= item.confirmations {
			// ***** FIX THIS: set verified for: item.payId
		} else {
			// if not yet at required confirmations, requeue at next possible block
			item.blockNumber = blockNumber + item.confirmations - reply.Confirmations
			heap.Push(pq, item)
		}
	}

}
