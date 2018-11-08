// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// various limiting constants
const (
	MaximumIssues = 100 // maximum allowable issues per block
)

// internal limiting constants
const (
	maximumPendingFreeIssues   = blockrecord.MaximumTransactions * 2
	maximumPendingPaidIssues   = blockrecord.MaximumTransactions * 2
	maximumPendingTransactions = blockrecord.MaximumTransactions * 16
)

// single transactions of any type
type transactionData struct {
	txId        merkle.Digest                 // transaction id
	transaction transactionrecord.Transaction // unpacked transaction
	packed      transactionrecord.Packed      // transaction bytes
}

// key: pay id
type transactionPaymentData struct {
	tx        *transactionData                       // record on this pay id
	payId     pay.PayId                              // for payment matching
	payments  []transactionrecord.PaymentAlternative // required payment
	expiresAt time.Time                              // only used in pending state
}

// key: pay id
type issuePaymentData struct {
	txs       []*transactionData                     // all records on this pay id
	payId     pay.PayId                              // for payment matching
	payments  []transactionrecord.PaymentAlternative // issue existing asset, or other records
	expiresAt time.Time                              // only used in pending state
}

// key: pay id
type issueFreeData struct {
	txs        []*transactionData     // all records on this pay id
	payId      pay.PayId              // for payment matching
	nonce      PayNonce               // only free issue, client nonce from successful try proof RPC
	difficulty *difficulty.Difficulty // only free issue, to test client nonce
	expiresAt  time.Time              // only used in pending state
}

type PaymentDetail struct {
	Currency currency.Currency // code number
	TxID     string            // tx id on currency blockchain
	Amounts  map[string]uint64 // address(Base58) → value(Satoshis)
}

type globalDataType struct {
	sync.RWMutex

	// to prevent fetch during critical operations
	enabled bool

	filename string

	log *logger.L

	background *background.T

	// separate verified pools
	verifiedTransactions map[pay.PayId]*transactionData  // normal transactions
	verifiedFreeIssues   map[pay.PayId]*issueFreeData    // so proof can be recreated
	verifiedPaidIssues   map[pay.PayId]*issuePaymentData // so block can be confirmed as a whole
	verifiedIndex        map[merkle.Digest]pay.PayId     // tx id → pay id

	// Link -> TxId to check for double spend
	inProgressLinks map[merkle.Digest]merkle.Digest

	// separate pending pools
	pendingTransactions map[pay.PayId]*transactionPaymentData
	pendingFreeIssues   map[pay.PayId]*issueFreeData
	pendingPaidIssues   map[pay.PayId]*issuePaymentData
	pendingIndex        map[merkle.Digest]pay.PayId // tx id → pay is

	pendingFreeCount int
	pendingPaidCount int

	// payments that are valid but have no pending record
	// ***** FIX THIS: need to expire
	orphanPayments map[pay.PayId]*PaymentDetail

	// set once during initialise
	initialised bool
}

// globals as a struct to allow lock
var globalData globalDataType

// create the cache
func Initialise(reservoirDataFile string) error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("reservoir")
	globalData.log.Info("starting…")

	globalData.inProgressLinks = make(map[merkle.Digest]merkle.Digest)

	globalData.verifiedTransactions = make(map[pay.PayId]*transactionData)
	globalData.verifiedFreeIssues = make(map[pay.PayId]*issueFreeData)
	globalData.verifiedPaidIssues = make(map[pay.PayId]*issuePaymentData)
	globalData.verifiedIndex = make(map[merkle.Digest]pay.PayId)

	globalData.pendingTransactions = make(map[pay.PayId]*transactionPaymentData)
	globalData.pendingFreeIssues = make(map[pay.PayId]*issueFreeData)
	globalData.pendingPaidIssues = make(map[pay.PayId]*issuePaymentData)
	globalData.pendingIndex = make(map[merkle.Digest]pay.PayId)

	globalData.pendingFreeCount = 0
	globalData.pendingPaidCount = 0

	globalData.orphanPayments = make(map[pay.PayId]*PaymentDetail)

	globalData.filename = reservoirDataFile

	// all data initialised
	globalData.initialised = true

	globalData.log.Debugf("load from file: %s", reservoirDataFile)
	globalData.Unlock()
	loadFromFile() // this uses locks in calls it makes
	Enable()
	globalData.Lock()

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
		&rebroadcaster{},
		&cleaner{},
	}

	globalData.background = background.Start(processes, nil)

	return nil
}

// stop all
func Finalise() error {

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()

	// save data
	saveToFile()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// for API to get status data
func ReadCounters() (int, int) {
	globalData.RLock()
	pending := len(globalData.pendingIndex)
	verified := len(globalData.verifiedIndex)
	globalData.RUnlock()
	return pending, verified
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

	_, ok := globalData.pendingIndex[txId]
	if ok {
		return StatePending
	}

	_, ok = globalData.verifiedIndex[txId]
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

	if nil == detail {
		globalData.log.Warn("payment was not provided")
		return false
	}

	globalData.log.Infof("detail: currency: %s, amounts: %#v", detail.Currency, detail.Amounts)

	// single transaction
	if entry, ok := globalData.pendingTransactions[payId]; ok {
		if !acceptablePayment(detail, entry.payments) {
			globalData.log.Warnf("failed check for txid: %s  payid: %s", detail.TxID, payId)
			return false
		}
		globalData.log.Infof("paid txid: %s  payid: %s", detail.TxID, payId)

		delete(globalData.pendingTransactions, payId)
		globalData.verifiedTransactions[payId] = entry.tx

		txId := entry.tx.txId

		delete(globalData.pendingIndex, txId)
		globalData.verifiedIndex[txId] = payId

		return true
	}

	// issue block
	if entry, ok := globalData.pendingPaidIssues[payId]; ok {
		if !acceptablePayment(detail, entry.payments) {
			globalData.log.Warnf("failed check for txid: %s  payid: %s", detail.TxID, payId)
			return false
		}
		globalData.log.Infof("paid txid: %s  payid: %s", detail.TxID, payId)

		globalData.pendingPaidCount -= len(entry.txs)
		delete(globalData.pendingPaidIssues, payId)
		globalData.verifiedPaidIssues[payId] = entry

		for _, tx := range entry.txs {
			txId := tx.txId

			delete(globalData.pendingIndex, txId)
			globalData.verifiedIndex[txId] = payId
		}

		return true
	}

	return false
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

// set verified if transaction found, otherwise preserv payment for later
func SetTransferVerified(payId pay.PayId, detail *PaymentDetail) {
	globalData.log.Infof("txid: %s  payid: %s", detail.TxID, payId)

	globalData.Lock()
	if !setVerified(payId, detail) {
		globalData.log.Debugf("orphan payment: txid: %s  payid: %s", detail.TxID, payId)
		globalData.orphanPayments[payId] = detail
	}
	globalData.Unlock()
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

// before calling Enable may need to run rescan to drop any
// invalidated transactions especially if the block height has changed
func Rescan() {
	globalData.Lock()
	defer globalData.Unlock()

	// pending

	for _, item := range globalData.pendingTransactions {
		rescanItem(item.tx)
	}
	for _, item := range globalData.pendingFreeIssues {
		for _, tx := range item.txs {
			rescanItem(tx)
		}
	}
	for _, item := range globalData.pendingPaidIssues {
		for _, tx := range item.txs {
			rescanItem(tx)
		}
	}

	// verified

	for _, tx := range globalData.verifiedTransactions {
		rescanItem(tx)
	}
	for _, item := range globalData.verifiedFreeIssues {
		for _, tx := range item.txs {
			rescanItem(tx)
		}
	}
	for _, item := range globalData.verifiedPaidIssues {
		for _, tx := range item.txs {
			rescanItem(tx)
		}
	}
}

func rescanItem(item *transactionData) {

	txId := item.txId

	// repack records to check signature is valid
	switch tx := item.transaction.(type) {

	case *transactionrecord.OldBaseData:
		logger.Panic("reservoir: rescan found: OldBaseData")

	case *transactionrecord.AssetData:
		logger.Panic("reservoir: rescan found: AssetData")

	case *transactionrecord.BitmarkIssue:
		if storage.Pool.Transactions.Has(txId[:]) {
			DeleteByTxId(txId)
		}

	case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
		tr := tx.(transactionrecord.BitmarkTransfer)
		link := tr.GetLink()
		linkOwner := ownership.OwnerOf(link)
		if nil == linkOwner {
			logger.Criticalf("missing transaction record for link: %v refererenced by tx: %+v", link, tx)
			logger.Panic("Transactions database is corrupt")
		}
		if !ownership.CurrentlyOwns(linkOwner, link) {
			DeleteByTxId(txId)
		}

	case *transactionrecord.BlockFoundation:
		logger.Panic("reservoir: rescan found: BlockFoundation")

	case *transactionrecord.BlockOwnerTransfer:
		link := tx.Link
		linkOwner := ownership.OwnerOf(link)
		if !ownership.CurrentlyOwns(linkOwner, link) {
			DeleteByTxId(txId)
		}

	default:
		globalData.log.Criticalf("reservoir rescan unhandled transaction: %v", tx)
		logger.Panicf("unhandled transaction: %v", tx)
	}
}

// remove a record using a transaction id
// note, remove one issue in a block removes the whole issue block
func DeleteByTxId(txId merkle.Digest) {
	if enabled() {
		logger.Panic("reservoir delete tx id when not locked")
	}
	if payId, ok := globalData.pendingIndex[txId]; ok {
		internalDelete(payId)
	}
	if payId, ok := globalData.verifiedIndex[txId]; ok {
		internalDelete(payId)
	}
}

// remove a record using a link id
func DeleteByLink(link merkle.Digest) {
	if enabled() {
		logger.Panic("reservoir delete link when not locked")
	}
	if txId, ok := globalData.inProgressLinks[link]; ok {
		if payId, ok := globalData.pendingIndex[txId]; ok {
			internalDelete(payId)
		}
		if payId, ok := globalData.verifiedIndex[txId]; ok {
			internalDelete(payId)
		}
	}
}

// delete all buffered transactions relating to a pay id
// (after it has been confirmed)
func internalDelete(payId pay.PayId) {

	// pending

	if entry, ok := globalData.pendingTransactions[payId]; ok {
		delete(globalData.pendingIndex, entry.tx.txId)
		if transfer, ok := entry.tx.transaction.(transactionrecord.BitmarkTransfer); ok {
			link := transfer.GetLink()
			delete(globalData.inProgressLinks, link)
		}
		delete(globalData.pendingTransactions, payId)
	}

	if entry, ok := globalData.pendingFreeIssues[payId]; ok {
		for _, tx := range entry.txs {
			delete(globalData.pendingIndex, tx.txId)
		}
		globalData.pendingFreeCount -= len(entry.txs)
		delete(globalData.pendingFreeIssues, payId)
	}

	if entry, ok := globalData.pendingPaidIssues[payId]; ok {
		for _, tx := range entry.txs {
			delete(globalData.pendingIndex, tx.txId)
		}
		globalData.pendingPaidCount -= len(entry.txs)
		delete(globalData.pendingPaidIssues, payId)
	}

	// verified

	if entry, ok := globalData.verifiedTransactions[payId]; ok {
		delete(globalData.verifiedIndex, entry.txId)
		if transfer, ok := entry.transaction.(transactionrecord.BitmarkTransfer); ok {
			link := transfer.GetLink()
			delete(globalData.inProgressLinks, link)
		}
		delete(globalData.verifiedTransactions, payId)
	}

	if entry, ok := globalData.verifiedFreeIssues[payId]; ok {
		for _, tx := range entry.txs {
			delete(globalData.verifiedIndex, tx.txId)
		}
		delete(globalData.verifiedFreeIssues, payId)
	}

	if entry, ok := globalData.verifiedPaidIssues[payId]; ok {
		for _, tx := range entry.txs {
			delete(globalData.verifiedIndex, tx.txId)
		}
		delete(globalData.verifiedPaidIssues, payId)
	}
}
