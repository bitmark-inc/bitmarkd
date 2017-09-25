// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
)

type itemData struct {
	txIds        []merkle.Digest
	links        []merkle.Digest                // links[i] corresponds to txIds[i]
	assetIds     []transactionrecord.AssetIndex // asset[i] index corresponds to txIds[i]
	transactions [][]byte                       // transactions[i] corresponds to txIds[i]
}

type unverifiedItem struct {
	*itemData
	nonce      PayNonce                               // only for issues
	difficulty *difficulty.Difficulty                 // only for issues
	payments   []transactionrecord.PaymentAlternative // currently only for transfers
}

type verifiedItem struct {
	*itemData
	link        merkle.Digest
	transaction []byte
	index       int // index of assetIds and transactions in an item
}

type PaymentDetail struct {
	Currency currency.Currency
	TxID     string
	Amounts  map[string]uint64
}

type globalDataType struct {
	sync.RWMutex
	enabled    bool
	log        *logger.L
	background *background.T
}

var globalData globalDataType

// create the cache
func Initialise() error {
	globalData.log = logger.New("reservoir")
	if globalData.log == nil {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	Enable()

	// start background process "rebroadcaster"
	var reb rebroadcaster
	reb.log = logger.New("rebroadcaster")
	if reb.log == nil {
		return fault.ErrInvalidLoggerChannel
	}

	processes := background.Processes{&reb}
	globalData.background = background.Start(processes, &globalData)

	return nil
}

// stop all
func Finalise() {
	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	Disable()

	globalData.background.Stop()

	globalData.log.Info("finished")
	globalData.log.Flush()
}

func ReadCounters() (int, int, []int) {
	n := []int{
		cache.Pool.PendingTransfer.Size(),
		cache.Pool.UnverifiedTxEntries.Size(),
	}
	return cache.Pool.UnverifiedTxIndex.Size(), cache.Pool.VerifiedTx.Size(), n
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

	_, ok := cache.Pool.UnverifiedTxIndex.Get(txId.String())
	if ok {
		return StatePending
	}

	_, ok = cache.Pool.VerifiedTx.Get(txId.String())
	if ok {
		return StateVerified
	}

	if storage.Pool.Transactions.Has(txId[:]) {
		return StateConfirmed
	}

	return StateUnknown
}

// move transaction(s) to verified cache
func setVerified(payId pay.PayId, detail *PaymentDetail) bool {
	val, ok := cache.Pool.UnverifiedTxEntries.Get(payId.String())
	if ok {
		entry := val.(*unverifiedItem)

		if nil != detail {
			globalData.log.Infof("detail: currency: %s, amounts: %#v", detail.Currency, detail.Amounts)
		}

		if nil != entry.payments {
			if !acceptablePayment(detail, entry.payments) {
				globalData.log.Warnf("failed check for txid: %s  payid: %s", detail.TxID, payId)
				return false
			}
			globalData.log.Infof("paid txid: %s  payid: %s", detail.TxID, payId)
		}

		var filter ProofFilter

		for i, txId := range entry.txIds {
			v := &verifiedItem{
				itemData:    entry.itemData,
				transaction: entry.transactions[i],
				index:       i,
			}
			if nil != entry.links {
				v.link = entry.links[i]
			}
			filter.Add(entry.transactions[i])

			cache.Pool.VerifiedTx.Put(txId.String(), v)
			cache.Pool.UnverifiedTxIndex.Delete(txId.String())
		}

		cache.Pool.ProofFilters.Put(payId.String(), filter)
		cache.Pool.UnverifiedTxEntries.Delete(payId.String())
	}

	return ok
}

// check that the incoming payment details match the stored payments records
func acceptablePayment(detail *PaymentDetail, payments []transactionrecord.PaymentAlternative) bool {
next_currency:
	for _, p := range payments {
		acceptable := true
		globalData.log.Infof("sv: payment: %#v", p)
		for _, item := range p {
			globalData.log.Infof("sv: item: %#v", item)
			if item.Currency != detail.Currency {
				continue next_currency
			}
			if detail.Amounts[item.Address] < item.Amount {
				acceptable = false
			}
		}
		if acceptable {
			return true
		}
	}
	return false
}

func SetTransferVerified(payId pay.PayId, detail *PaymentDetail) {
	globalData.log.Infof("txid: %s  payid: %s", detail.TxID, payId)

	if !setVerified(payId, detail) {
		globalData.log.Infof("orphan payment: txid: %s  payid: %s", detail.TxID, payId)
		cache.Pool.OrphanPayment.Put(payId.String(), detail)
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
	if enabled() {
	loop:
		for key, val := range cache.Pool.VerifiedTx.Items() {
			var txId merkle.Digest
			fmt.Sscan(key, &txId)
			data := val.(*verifiedItem)

			txIds = append(txIds, txId)
			txData = append(txData, data.transaction)
			totalBytes += len(data.transaction)
			n += 1
			if n >= count {
				break loop
			}
		}
	}

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

func enabled() bool {
	globalData.RLock()
	defer globalData.RUnlock()
	return globalData.enabled
}

// remove a record using a transaction id
func DeleteByTxId(txId merkle.Digest) {
	if enabled() {
		logger.Panic("reservoir delete tx id when not locked")
	}

	if val, ok := cache.Pool.UnverifiedTxIndex.Get(txId.String()); ok {
		payId := val.(pay.PayId)
		internalDelete(payId)
	}
	if val, ok := cache.Pool.VerifiedTx.Get(txId.String()); ok {
		item := val.(*verifiedItem)
		link := item.link
		cache.Pool.VerifiedTx.Delete(txId.String())
		cache.Pool.PendingTransfer.Delete(link.String())
	}
}

// remove a record using a link id
func DeleteByLink(link merkle.Digest) {
	if enabled() {
		logger.Panic("reservoir delete link when not locked")
	}
	if val, ok := cache.Pool.PendingTransfer.Get(link.String()); ok {
		txId := val.(merkle.Digest)
		if val, ok := cache.Pool.UnverifiedTxIndex.Get(txId.String()); ok {
			payId := val.(pay.PayId)
			internalDelete(payId)
		}
		if val, ok := cache.Pool.VerifiedTx.Get(txId.String()); ok {
			item := val.(*verifiedItem)
			link := item.link
			cache.Pool.VerifiedTx.Delete(txId.String())
			cache.Pool.PendingTransfer.Delete(link.String())
		}
	}
}

// delete unverified transactions
func internalDelete(payId pay.PayId) {
	val, ok := cache.Pool.UnverifiedTxEntries.Get(payId.String())
	if ok {
		entry := val.(*unverifiedItem)
		for i, txId := range entry.txIds {
			cache.Pool.UnverifiedTxIndex.Delete(txId.String())
			cache.Pool.VerifiedTx.Delete(txId.String())
			if nil != entry.links {
				link := entry.links[i]
				cache.Pool.PendingTransfer.Delete(link.String())
			}
		}
		cache.Pool.UnverifiedTxEntries.Delete(payId.String())
	}
}
