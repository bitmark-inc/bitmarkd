// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"path"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/account"
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

// the cache file
const reservoirFile = "reservoir.cache"

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

// PaymentDetail - a payment record for a single currency
type PaymentDetail struct {
	Currency currency.Currency // code number
	TxID     string            // tx id on currency blockchain
	Amounts  map[string]uint64 // address(Base58) → value(Satoshis)
}

// track the shares
type spendKey struct {
	owner [64]byte
	share merkle.Digest
}

// Handles - storage handles used when restore from cache file
type Handles struct {
	Assets            storage.Handle
	BlockOwnerPayment storage.Handle
	Blocks            storage.Handle
	Transactions      storage.Handle
	OwnerTx           storage.Handle
	OwnerData         storage.Handle
	Share             storage.Handle
	ShareQuantity     storage.Handle
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

	// tracking the shares
	spend map[spendKey]uint64

	// set once during initialise
	initialised bool

	handles Handles
}

// globals as a struct to allow lock
var globalData globalDataType

func (g *globalDataType) StoreTransfer(transfer transactionrecord.BitmarkTransfer) (*TransferInfo, bool, error) {
	return storeTransfer(
		transfer,
		g.handles.Transactions,
		g.handles.OwnerTx,
		g.handles.OwnerData,
		g.handles.BlockOwnerPayment,
	)
}

func (g *globalDataType) StoreIssues(issues []*transactionrecord.BitmarkIssue) (*IssueInfo, bool, error) {
	return storeIssues(
		issues,
		g.handles.Assets,
		g.handles.BlockOwnerPayment,
	)
}

func (g *globalDataType) TryProof(payID pay.PayId, clientNonce []byte) TrackingStatus {
	return tryProof(payID, clientNonce)
}

func (g *globalDataType) TransactionStatus(txID merkle.Digest) TransactionState {
	return transactionStatus(txID)
}

func (g *globalDataType) ShareBalance(owner *account.Account, startSharedID merkle.Digest, count int) ([]BalanceInfo, error) {
	return shareBalance(owner, startSharedID, count, g.handles.ShareQuantity)
}

func (g *globalDataType) StoreGrant(grant *transactionrecord.ShareGrant) (*GrantInfo, bool, error) {
	return storeGrant(
		grant,
		g.handles.ShareQuantity,
		g.handles.Share,
		g.handles.OwnerData,
		g.handles.BlockOwnerPayment,
		g.handles.Transactions,
	)
}

func (g *globalDataType) StoreSwap(swap *transactionrecord.ShareSwap) (*SwapInfo, bool, error) {
	return storeSwap(
		swap,
		g.handles.ShareQuantity,
		g.handles.Share,
		g.handles.OwnerData,
		g.handles.BlockOwnerPayment,
	)
}

type Reservoir interface {
	StoreTransfer(transactionrecord.BitmarkTransfer) (*TransferInfo, bool, error)
	StoreIssues(issues []*transactionrecord.BitmarkIssue) (*IssueInfo, bool, error)
	TryProof(pay.PayId, []byte) TrackingStatus
	TransactionStatus(merkle.Digest) TransactionState
	ShareBalance(*account.Account, merkle.Digest, int) ([]BalanceInfo, error)
	StoreGrant(*transactionrecord.ShareGrant) (*GrantInfo, bool, error)
	StoreSwap(swap *transactionrecord.ShareSwap) (*SwapInfo, bool, error)
}

func Get() Reservoir {
	if globalData.enabled {
		return &globalData
	}
	return nil
}

// Initialise - create the cache
func Initialise(cacheDirectory string, handles Handles) error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
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

	globalData.spend = make(map[spendKey]uint64)

	globalData.enabled = true

	globalData.filename = path.Join(cacheDirectory, reservoirFile)

	// all data initialised
	globalData.initialised = true

	globalData.log.Debugf("load from file: %s", globalData.filename)

	// start background processes
	globalData.log.Info("start background…")

	globalData.handles = handles

	processes := background.Processes{
		&rebroadcaster{},
		&cleaner{},
	}

	globalData.background = background.Start(processes, nil)

	return nil
}

// Finalise - stop all background processes
func Finalise() error {

	if !globalData.initialised {
		return fault.NotInitialised
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

// ReadCounters - for API to get status data
func ReadCounters() (int, int) {
	globalData.RLock()
	pending := len(globalData.pendingIndex)
	verified := len(globalData.verifiedIndex)
	globalData.RUnlock()
	return pending, verified
}

// TransactionState - status enumeration
type TransactionState int

// list of all states
const (
	StateUnknown   TransactionState = iota
	StatePending   TransactionState = iota
	StateVerified  TransactionState = iota
	StateConfirmed TransactionState = iota
)

// String - string representation of a transaction state
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

// transactionStatus - get status of a transaction
func transactionStatus(txId merkle.Digest) TransactionState {
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

	if globalData.handles.Transactions.Has(txId[:]) {
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
			globalData.log.Warnf("single transaction failed check for txid: %s  payid: %s", detail.TxID, payId)
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
			globalData.log.Warnf("issue block failed check for txid: %s  payid: %s", detail.TxID, payId)
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
		for _, item := range p {
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

// SetTransferVerified - set verified if transaction found, otherwise preserv payment for later
func SetTransferVerified(payId pay.PayId, detail *PaymentDetail) {
	globalData.log.Infof("txid: %s  payid: %s", detail.TxID, payId)

	globalData.Lock()
	if !setVerified(payId, detail) {
		globalData.log.Debugf("orphan payment: txid: %s  payid: %s", detail.TxID, payId)
		globalData.orphanPayments[payId] = detail
	}
	globalData.Unlock()
}

// Disable - lock down to prevent proofer from getting data
func Disable() {
	globalData.Lock()
	globalData.enabled = false
	globalData.Unlock()
}

// Enable - allow proofer to run again
func Enable() {
	globalData.Lock()
	globalData.enabled = true
	globalData.Unlock()
}

// ClearSpend - reset spend map
func ClearSpend() {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.enabled {
		logger.Panic("reservoir clear spend when not locked")
	}

	globalData.spend = make(map[spendKey]uint64)
}

// Rescan - before calling Enable may need to run rescan to drop any
// invalidated transactions especially if the block height has changed
func Rescan() {
	globalData.Lock()
	defer globalData.Unlock()

	//empty the spend map
	globalData.spend = make(map[spendKey]uint64)

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
		// should never be in the memory pool - so panic
		logger.Panic("reservoir: rescan found: OldBaseData")

	case *transactionrecord.AssetData:
		// should never be in the memory pool - so panic
		logger.Panic("reservoir: rescan found: AssetData")

	case *transactionrecord.BitmarkIssue:
		if storage.Pool.Transactions.Has(txId[:]) {
			internalDeleteByTxId(txId)
		}

	case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
		tr := tx.(transactionrecord.BitmarkTransfer)
		link := tr.GetLink()
		_, linkOwner := ownership.OwnerOf(nil, link)
		if nil == linkOwner || !ownership.CurrentlyOwns(nil, linkOwner, link, storage.Pool.OwnerTxIndex) {
			internalDeleteByTxId(txId)
		}

	case *transactionrecord.BlockFoundation:
		// should never be in the memory pool - so panic
		logger.Panic("reservoir: rescan found: BlockFoundation")

	case *transactionrecord.BlockOwnerTransfer:
		link := tx.Link
		_, linkOwner := ownership.OwnerOf(nil, link)
		if nil == linkOwner || !ownership.CurrentlyOwns(nil, linkOwner, link, storage.Pool.OwnerTxIndex) {
			internalDeleteByTxId(txId)
		}

	case *transactionrecord.BitmarkShare:
		link := tx.Link
		_, linkOwner := ownership.OwnerOf(nil, link)
		if nil == linkOwner || !ownership.CurrentlyOwns(nil, linkOwner, link, storage.Pool.OwnerTxIndex) {
			internalDeleteByTxId(txId)
		}

	case *transactionrecord.ShareGrant:
		_, err := CheckGrantBalance(nil, tx, storage.Pool.ShareQuantity)
		if nil != err {
			internalDeleteByTxId(txId)
		} else {
			k := makeSpendKey(tx.Owner, tx.ShareId)
			globalData.spend[k] += tx.Quantity
		}

	case *transactionrecord.ShareSwap:
		_, _, err := CheckSwapBalances(nil, tx, storage.Pool.ShareQuantity)
		if nil != err {
			internalDeleteByTxId(txId)
		} else {
			k := makeSpendKey(tx.OwnerOne, tx.ShareIdOne)
			globalData.spend[k] += tx.QuantityOne
			k = makeSpendKey(tx.OwnerTwo, tx.ShareIdTwo)
			globalData.spend[k] += tx.QuantityTwo
		}

	default:
		// undefined data in the memory pool - so panic
		globalData.log.Criticalf("reservoir rescan unhandled transaction: %v", tx)
		logger.Panicf("unhandled transaction: %v", tx)
	}
}

// DeleteByTxId - remove a record using a transaction id
// note, remove one issue in a block removes the whole issue block
func DeleteByTxId(txId merkle.Digest) {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.enabled {
		logger.Panic("reservoir delete tx id when not locked")
	}

	internalDeleteByTxId(txId)
}

// non-locking version of above
func internalDeleteByTxId(txId merkle.Digest) {
	if payId, ok := globalData.pendingIndex[txId]; ok {
		internalDelete(payId)
	}
	if payId, ok := globalData.verifiedIndex[txId]; ok {
		internalDelete(payId)
	}
}

// DeleteByLink - remove a record using a link id
func DeleteByLink(link merkle.Digest) {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.enabled {
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
