// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"time"
)

// result returned by store transfer
type TransferInfo struct {
	Id       pay.PayId
	TxId     merkle.Digest
	Packed   []byte
	Payments []*transactionrecord.Payment
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
	payId := pay.NewPayId([][]byte{packedTransfer})

	txId := verifyResult.txId
	link := transfer.Link
	if txId == link {
		// reject any transaction that links to itself
		// this should never occur, but protect agains this situuation
		return nil, false, fault.ErrTransactionLinksToSelf
	}

	previousTransfer := verifyResult.previousTransfer
	ownerData := verifyResult.ownerData

	payments := payment.GetPayments(ownerData, previousTransfer)

	result := &TransferInfo{
		Id:       payId,
		TxId:     txId,
		Packed:   packedTransfer,
		Payments: payments,
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

	expiresAt := time.Now().Add(constants.ReservoirTimeout)

	// create index and pending entries
	globalData.unverified.index[txId] = payId
	globalData.pendingTransfer[link] = txId

	// save transactions
	entry := &unverifiedItem{
		txIds:        []merkle.Digest{txId},
		links:        []merkle.Digest{link},
		transactions: [][]byte{packedTransfer},
		payments:     payments,
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
// ensure lock is held before calling
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
