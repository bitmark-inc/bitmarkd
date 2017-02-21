// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"container/heap"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"time"
)

// accept a new payment to monitor
func QueueItem(payId reservoir.PayId, txId string, confirmations uint64, payments []*transactionrecord.Payment) bool {
	globalData.itemQueue <- &priorityItem{
		payId:         payId,
		txId:          txId,
		confirmations: confirmations,
		//blockNumber:   globalData.latestBlockNumber + confirmations,
		blockNumber: 110, // ***** FIX THIS: temporary for debugging
		payments:    payments,
	}

	// ***** FIX THIS: to return a proper status
	// ***** FIX THIS: need to validate the transaction initially
	return true // ***** FIX THIS: assume success
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
		// case blockNumber := <-state.blockQueue:
		// 	log.Infof("block number: %d", blockNumber)
		// 	//state.latestBlockNumber := blockNumber
		// 	process(log, pq, blockNumber, state.verifier)
		case item := <-state.itemQueue:
			//item.blockNumber = state.latestBlockNumber + item.confirmations
			heap.Push(pq, item)

		case <-time.After(60 * time.Second):
			var blockNumber uint64
			err := bitcoinCall("getblockcount", []interface{}{}, &blockNumber)
			if nil != err {
				continue loop
			}
			log.Infof("block number: %d", blockNumber)
			state.latestBlockNumber = blockNumber
			process(log, pq, blockNumber, state.verifier)
		}
	}
}

// process all items <= block number
func process(log *logger.L, pq *priorityQueue, blockNumber uint64, verifier chan<- reservoir.PayId) {

	const (
		OP_RETURN       = "6a"   // plain op code
		OP_RETURN_COUNT = "6a30" // op code with 48 byte parameter
	)

loop:
	for pq.Len() > 0 {
		item := heap.Pop(pq).(*priorityItem)
		log.Infof("check payId: %s ", item.payId)

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
			item.blockNumber = blockNumber + item.confirmations
			heap.Push(pq, item) // retry at a later time
			continue loop       // ***** FIX THIS: need to limit the number of retries
		}

		payIdString := item.payId.String()
		amounts := make(map[string]uint64)
		ok := false
		for i, vout := range reply.Vout {
			log.Infof("vout[%d]: %v ", i, vout)
			if OP_RETURN == vout.ScriptPubKey.Hex[0:2] && vout.ScriptPubKey.Hex[2:] == payIdString {
				ok = true
				continue
			}
			if OP_RETURN_COUNT == vout.ScriptPubKey.Hex[0:4] && vout.ScriptPubKey.Hex[4:] == payIdString {
				ok = true
				continue
			}
			if 1 == len(vout.ScriptPubKey.Addresses) {
				amounts[vout.ScriptPubKey.Addresses[0]] += convertToSatoshi(vout.Value)
			}
		}
		if !ok {
			log.Errorf("no payId: %s in tx: %s", item.payId, item.txId)
			continue loop // item is dropped from the heap
		}
		for _, item := range item.payments {
			v := amounts[item.Address]
			if v >= item.Amount {
				amounts[item.Address] -= item.Amount
			} else {
				log.Errorf("insufficient payment t: %s  need: %d  have: %d", item.Address, item.Amount, v)
				ok = false
				break
			}
		}
		if !ok {
			continue loop // item is dropped from the heap
		}

		if reply.Confirmations >= item.confirmations {
			log.Infof("confirming payId: %s in tx: %s", item.payId, item.txId)
			// send the transaction block to verifier
			verifier <- item.payId
		} else {
			log.Infof("insufficient confirmations for payId: %s in tx: %s", item.payId, item.txId)
			// if not yet at required confirmations, requeue at next possible block
			item.blockNumber = blockNumber + item.confirmations - reply.Confirmations
			heap.Push(pq, item)
		}
	}

}
