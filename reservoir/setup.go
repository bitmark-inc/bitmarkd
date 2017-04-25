// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
	"time"
)

// globals
type globalDataType struct {
	sync.RWMutex
	log        *logger.L
	enabled    bool
	unverified unverifiedEntry
	verified   map[merkle.Digest]*verifiedItem

	// indexed by link so that duplicate transfers can be detected
	// data is the tx id so that the same transfer repeated can be distinguished
	// from an invalid duplicate transfer
	pendingTransfer map[merkle.Digest]merkle.Digest

	verifier      verifierData
	rebroadcaster rebroadcaster
	background    *background.T
}

type unverifiedEntry struct {
	entries map[pay.PayId]*unverifiedItem
	index   map[merkle.Digest]pay.PayId
}

type itemData struct {
	txIds        []merkle.Digest
	links        []merkle.Digest                // links[i] corresponds to txIds[i]
	assetIds     []transactionrecord.AssetIndex // asset[i] index corresponds to txIds[i]
	transactions [][]byte                       // transactions[i] corresponds to txIds[i]
}

type unverifiedItem struct {
	*itemData
	nonce      PayNonce                     // only for issues
	difficulty *difficulty.Difficulty       // only for issues
	payments   []*transactionrecord.Payment // currently only for transfers
	expires    time.Time
}

type verifiedItem struct {
	link        merkle.Digest
	transaction []byte
	data        *itemData // point to the item struct
	index       int       // index of assetIds and transactions in an item
}

// background data
type verifierData struct {
	log *logger.L
}

type rebroadcaster struct {
	log *logger.L
}

// gobal storage
var globalData globalDataType

// create the cache
func Initialise() error {

	globalData.Lock()
	defer globalData.Unlock()

	globalData.log = logger.New("reservoir")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.unverified.entries = make(map[pay.PayId]*unverifiedItem)
	globalData.unverified.index = make(map[merkle.Digest]pay.PayId)
	globalData.verified = make(map[merkle.Digest]*verifiedItem)
	globalData.pendingTransfer = make(map[merkle.Digest]merkle.Digest)

	globalData.enabled = true

	globalData.verifier.log = logger.New("reservoir-verifier")
	if nil == globalData.verifier.log {
		return fault.ErrInvalidLoggerChannel
	}

	globalData.rebroadcaster.log = logger.New("rebroadcaster")
	if nil == globalData.rebroadcaster.log {
		return fault.ErrInvalidLoggerChannel
	}

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	var processes = background.Processes{
		&globalData.verifier,
		&globalData.rebroadcaster,
	}

	globalData.background = background.Start(processes, &globalData)

	return nil
}

// stop all
func Finalise() {

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	globalData.enabled = false

	// stop background
	globalData.background.Stop()

	globalData.log.Info("finished")
	globalData.log.Flush()
}

// read counter
func ReadCounters() (int, int, []int) {
	n := []int{
		len(globalData.pendingTransfer),
		len(globalData.unverified.entries),
	}
	return len(globalData.unverified.index), len(globalData.verified), n
}

// status
type TransactionState int

const (
	StateUnknown   TransactionState = iota
	StatePending   TransactionState = iota
	StateVerified  TransactionState = iota
	StateConfirmed TransactionState = iota
)

func (state TransactionState) String() string {
	switch state {
	case StateUnknown:
		return "Unknown"
	case StatePending:
		return "Pending"
	case StateVerified:
		return "Verified"
	case StateConfirmed:
		return "Confirmed"
	default:
		return "Unknown"
	}
}

// get status of a transaction
func TransactionStatus(txId merkle.Digest) TransactionState {
	globalData.RLock()
	defer globalData.RUnlock()

	_, ok := globalData.unverified.index[txId]
	if ok {
		return StatePending
	}

	_, ok = globalData.verified[txId]
	if ok {
		return StateVerified
	}

	if storage.Pool.Transactions.Has(txId[:]) {
		return StateConfirmed
	}

	return StateUnknown
}

// move transaction(s) to verified cache
// must hold lock before calling this
func setVerified(payId pay.PayId) {
	entry, ok := globalData.unverified.entries[payId]
	if ok {
		// move the record
		for i, txId := range entry.txIds {
			v := &verifiedItem{
				data:        entry.itemData,
				transaction: entry.transactions[i],
				index:       i,
			}
			if nil != entry.links {
				v.link = entry.links[i]
			}
			globalData.verified[txId] = v
			delete(globalData.unverified.index, txId)
		}
		delete(globalData.unverified.entries, payId)
	}
}

// fetch a series of verified transactions
func FetchVerified(count int) ([]merkle.Digest, []transactionrecord.Packed, int, error) {
	if count <= 0 {
		return nil, nil, 0, fault.ErrInvalidCount
	}

	txIds := make([]merkle.Digest, 0, count)
	txData := make([]transactionrecord.Packed, 0, count)

	n := 0
	totalBytes := 0
	globalData.RLock()
	if globalData.enabled {
		for txId, data := range globalData.verified {
			txIds = append(txIds, txId)
			txData = append(txData, data.transaction)
			totalBytes += len(data.transaction)
			n += 1
			if n >= count {
				break
			}
		}
	}
	globalData.RUnlock()
	return txIds, txData, totalBytes, nil
}

// lock down to prevent proofer from getting data
func Disable() {
	globalData.Lock()
	globalData.enabled = false
	globalData.Unlock()
}

// allow proofer to run again
func Enable() {
	globalData.Lock()
	globalData.enabled = true
	globalData.Unlock()
}

// remove a record using a transaction id
func DeleteByTxId(txId merkle.Digest) {
	globalData.Lock()
	if globalData.enabled {
		fault.Panic("reservoir delete tx id when not locked")
	}
	if payId, ok := globalData.unverified.index[txId]; ok {
		internalDelete(payId)
	}
	if v, ok := globalData.verified[txId]; ok {
		link := v.link
		delete(globalData.verified, txId)
		delete(globalData.pendingTransfer, link)
	}
	globalData.Unlock()
}

// remove a record using a link id
func DeleteByLink(link merkle.Digest) {
	globalData.Lock()
	if globalData.enabled {
		fault.Panic("reservoir delete link when not locked")
	}
	if txId, ok := globalData.pendingTransfer[link]; ok {
		if payId, ok := globalData.unverified.index[txId]; ok {
			internalDelete(payId)
		}
		if v, ok := globalData.verified[txId]; ok {
			link := v.link
			delete(globalData.verified, txId)
			delete(globalData.pendingTransfer, link)
		}
	}
	globalData.Unlock()
}

// hold lock before calling
// delete unverified transactions
func internalDelete(payId pay.PayId) {
	entry, ok := globalData.unverified.entries[payId]
	if ok {
		for i, txId := range entry.txIds {
			delete(globalData.unverified.index, txId)
			delete(globalData.verified, txId)
			if nil != entry.links {
				link := entry.links[i]
				delete(globalData.pendingTransfer, link)
			}
		}
		delete(globalData.unverified.entries, payId)
	}
}
