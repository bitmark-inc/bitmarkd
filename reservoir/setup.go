// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
	"time"
)

const (
	expiryTime    = 2 * time.Hour
	maximumIssues = 100 // allowed issues in a single submission
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

	expiry     expiryData
	background *background.T
}

type unverifiedEntry struct {
	entries map[PayId]*unverifiedItem
	index   map[merkle.Digest]PayId
}

type unverifiedItem struct {
	txIds        []merkle.Digest
	links        []merkle.Digest
	transactions [][]byte
	expires      time.Time
}

type verifiedItem struct {
	link        merkle.Digest
	transaction []byte
}

// expiry background
type expiryData struct {
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

	globalData.unverified.entries = make(map[PayId]*unverifiedItem)
	globalData.unverified.index = make(map[merkle.Digest]PayId)
	globalData.verified = make(map[merkle.Digest]*verifiedItem)
	globalData.pendingTransfer = make(map[merkle.Digest]merkle.Digest)

	globalData.enabled = true

	globalData.expiry.log = logger.New("reservoir-expiry")
	if nil == globalData.expiry.log {
		return fault.ErrInvalidLoggerChannel
	}

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	var processes = background.Processes{
		&globalData.expiry,
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

// result returned by store issues
type IssueInfo struct {
	Id     PayId
	TxIds  []merkle.Digest
	Packed []byte
}

// store packed record(s) in the Unverified table
//
// return payment id and a duplicate flag
//
// for duplicate to be true all transactions must all match exactly to a
// previous set - this is to allow for multiple submission from client
// without receiving a duplicate transaction error
func StoreIssues(issues []*transactionrecord.BitmarkIssue) (*IssueInfo, bool, error) {

	count := len(issues)
	if count > maximumIssues {
		return nil, false, fault.ErrTooManyItemsToProcess
	} else if 0 == count {
		return nil, false, fault.ErrMissingParameters
	}

	// critical code - prevent overlapping blocks of issues
	globalData.Lock()
	defer globalData.Unlock()

	// individual packed issues
	separated := make([][]byte, 100)

	// all the tx id corresponding to separated
	txIds := make([]merkle.Digest, count)

	// this flags in already stored issues
	// used to flags an error if pay id is different
	// as this would be an overlapping block of issues
	duplicate := false

	// verify each transaction
	for i, issue := range issues {

		// validate issue record
		packedIssue, err := issue.Pack(issue.Owner)
		if nil != err {
			return nil, false, err
		}

		if !asset.Exists(issue.AssetIndex) {
			return nil, false, fault.ErrAssetNotFound
		}

		txId := packedIssue.MakeLink()

		// an unverified issue tag the block as possible duplicate
		// (if pay id matched later)
		if _, ok := globalData.unverified.index[txId]; ok {
			// if duplicate, activate pay id check
			duplicate = true
		}

		// a single verified issue fails the whole block
		if _, ok := globalData.verified[txId]; ok {
			return nil, false, fault.ErrTransactionAlreadyExists
		}
		// a single confirmed issue fails the whole block
		if storage.Pool.Transactions.Has(txId[:]) {
			return nil, false, fault.ErrTransactionAlreadyExists
		}

		// accumulate the data
		txIds[i] = txId
		separated[i] = packedIssue

	}

	// compute pay id
	payId := NewPayId(separated)

	result := &IssueInfo{
		Id:     payId,
		TxIds:  txIds,
		Packed: bytes.Join(separated, []byte{}),
	}

	// if already seen just return pay id
	if _, ok := globalData.unverified.entries[payId]; ok {
		globalData.log.Debugf("duplicate pay id: %s", payId)
		return result, true, nil
	}

	// if duplicates were detected, but duplicates were present
	// then it is an error
	if duplicate {
		globalData.log.Debugf("overlapping pay id: %s", payId)
		return nil, false, fault.ErrTransactionAlreadyExists
	}

	globalData.log.Infof("creating pay id: %s", payId)

	expiresAt := time.Now().Add(expiryTime)

	// create index entries
	for _, txId := range txIds {
		globalData.unverified.index[txId] = payId
	}

	// save transactions
	entry := &unverifiedItem{
		txIds:        txIds,
		links:        nil,
		transactions: separated,
		expires:      expiresAt,
	}
	//copy(entry.txIds, txIds)
	//copy(entry.transactions, transactions)

	globalData.unverified.entries[payId] = entry

	return result, false, nil
}

// result returned by store transfer
type TransferInfo struct {
	Id               PayId
	TxId             merkle.Digest
	Packed           []byte
	PreviousTransfer *transactionrecord.BitmarkTransfer
	OwnerData        []byte
}

// store a single transfer
func StoreTransfer(transfer *transactionrecord.BitmarkTransfer) (*TransferInfo, bool, error) {

	// critical code - prevent overlapping blocks of transactions
	globalData.Lock()
	defer globalData.Unlock()

	verifyResult, duplicate, err := verifyTransfer(transfer)
	if nil != err {
		return nil, false, err
	}

	// compute pay id
	packedTransfer := verifyResult.packedTransfer
	payId := NewPayId([][]byte{packedTransfer})

	txId := verifyResult.txId
	link := transfer.Link
	if txId == link {
		// reject any transaction that links to itself
		// this should never occur, but protect agains this situuation
		return nil, false, fault.ErrTransactionLinksToSelf
	}

	result := &TransferInfo{
		Id:               payId,
		TxId:             txId,
		Packed:           packedTransfer,
		PreviousTransfer: verifyResult.previousTransfer,
		OwnerData:        verifyResult.ownerData,
	}

	// if already seen just return pay id
	if _, ok := globalData.unverified.entries[payId]; ok {
		return result, true, nil
	}

	// if duplicates were detected, but different duplicates were present
	// then it is an error
	if duplicate {
		return nil, true, fault.ErrTransactionAlreadyExists
	}

	expiresAt := time.Now().Add(expiryTime)

	// create index and pending entries
	globalData.unverified.index[txId] = payId
	globalData.pendingTransfer[link] = txId

	// save transactions
	entry := &unverifiedItem{
		txIds:        []merkle.Digest{txId},
		links:        []merkle.Digest{link},
		transactions: [][]byte{packedTransfer},
		expires:      expiresAt,
	}

	globalData.unverified.entries[payId] = entry

	return result, false, nil
}

// returned data from veriftyTransfer
type verifiedInfo struct {
	txId             merkle.Digest
	packedTransfer   []byte
	previousTransfer *transactionrecord.BitmarkTransfer
	ownerData        []byte
}

// verify that a transfer is ok
func verifyTransfer(arguments *transactionrecord.BitmarkTransfer) (*verifiedInfo, bool, error) {

	// find the current owner via the link
	previousPacked := storage.Pool.Transactions.Get(arguments.Link[:])
	if nil == previousPacked {
		return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
	}

	previousTransaction, _, err := transactionrecord.Packed(previousPacked).Unpack()
	if nil != err {
		return nil, false, err
	}

	var currentOwner *account.Account
	var previousTransfer *transactionrecord.BitmarkTransfer

	switch tx := previousTransaction.(type) {
	case *transactionrecord.BitmarkIssue:
		currentOwner = tx.Owner

	case *transactionrecord.BitmarkTransfer:
		currentOwner = tx.Owner
		previousTransfer = tx

	default:
		return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
	}

	// pack transfer and check signature
	packedTransfer, err := arguments.Pack(currentOwner)
	if nil != err {
		return nil, false, err
	}

	// transfer identifier and check for duplicate
	txId := packedTransfer.MakeLink()

	// check if this transfer was already received
	_, okP := globalData.pendingTransfer[arguments.Link]
	_, okU := globalData.unverified.index[txId]
	duplicate := false
	if okU && okP {
		// if both then it is a possible duplicate
		// (depends on later pay id check)
		duplicate = true
	} else if okU || okP {
		// not an exact match - must be a double transfer
		return nil, false, fault.ErrDoubleTransferAttempt
	}

	// a single verified transfer fails the whole block
	if _, ok := globalData.verified[txId]; ok {
		return nil, false, fault.ErrTransactionAlreadyExists
	}
	// a single confirmed transfer fails the whole block
	if storage.Pool.Transactions.Has(txId[:]) {
		return nil, false, fault.ErrTransactionAlreadyExists
	}

	// log.Infof("packed transfer: %x", packedTransfer)
	// log.Infof("id: %v", txId)

	// get count for current owner record
	// to make sure that the record has not already been transferred
	dKey := append(currentOwner.Bytes(), arguments.Link[:]...)
	// log.Infof("dKey: %x", dKey)
	dCount := storage.Pool.OwnerDigest.Get(dKey)
	if nil == dCount {
		return nil, false, fault.ErrDoubleTransferAttempt
	}

	// get ownership data
	oKey := append(currentOwner.Bytes(), dCount...)
	// log.Infof("oKey: %x", oKey)
	ownerData := storage.Pool.Ownership.Get(oKey)
	if nil == ownerData {
		return nil, false, fault.ErrDoubleTransferAttempt
	}
	// log.Infof("ownerData: %x", ownerData)

	result := &verifiedInfo{
		txId:             txId,
		packedTransfer:   packedTransfer,
		previousTransfer: previousTransfer,
		ownerData:        ownerData,
	}
	return result, duplicate, nil
}

// move transaction(s) to verified cache
func SetVerified(payId PayId) {
	globalData.Lock()
	entry, ok := globalData.unverified.entries[payId]
	if ok {
		// move the record
		for i, txId := range entry.txIds {
			v := &verifiedItem{
				transaction: entry.transactions[i],
			}
			if nil != entry.links {
				v.link = entry.links[i]
			}
			globalData.verified[txId] = v
			delete(globalData.unverified.index, txId)
		}
		delete(globalData.unverified.entries, payId)
	}
	globalData.Unlock()
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
func Lock() {
	globalData.Lock()
	globalData.enabled = false
	globalData.Unlock()
}

// allow proofer to run again
func Unlock() {
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
func internalDelete(payId PayId) {
	entry, ok := globalData.unverified.entries[payId]
	if ok {
		for i, txId := range entry.txIds {
			delete(globalData.unverified.index, txId)
			delete(globalData.verified, txId)
			link := entry.links[i]
			delete(globalData.pendingTransfer, link)
		}
		delete(globalData.unverified.entries, payId)
	}
}
