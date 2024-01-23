// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// TransferInfo - result returned by store transfer
type TransferInfo struct {
	Id        pay.PayId
	TxId      merkle.Digest
	IssueTxId merkle.Digest
	Packed    []byte
	Payments  []transactionrecord.PaymentAlternative
}

// returned data from verifyTransfer
type verifiedTransferInfo struct {
	txId                merkle.Digest
	packed              []byte
	previousTransfer    transactionrecord.BitmarkTransfer
	issueTxId           merkle.Digest
	transferBlockNumber uint64
	issueBlockNumber    uint64
}

// storeTransfer - verify and store a transfer request
func storeTransfer(
	transfer transactionrecord.BitmarkTransfer,
	transactionHandle storage.Handle,
	ownerTxHandle storage.Handle,
	ownerDataHandle storage.Handle,
	blockOwnerPaymentHandle storage.Handle,
) (*TransferInfo, bool, error) {
	if transactionHandle == nil || ownerTxHandle == nil || ownerDataHandle == nil || blockOwnerPaymentHandle == nil {
		return nil, false, fault.NilPointer
	}

	globalData.Lock()
	defer globalData.Unlock()

	verifyResult, duplicate, err := verifyTransfer(transfer, transactionHandle, ownerTxHandle, ownerDataHandle)
	if err != nil {
		return nil, false, err
	}

	// compute pay id
	packedTransfer := verifyResult.packed
	payId := pay.NewPayId([][]byte{packedTransfer})

	txId := verifyResult.txId

	previousTransfer := verifyResult.previousTransfer

	payments := getPayments(verifyResult.transferBlockNumber, verifyResult.issueBlockNumber, previousTransfer, blockOwnerPaymentHandle)

	result := &TransferInfo{
		Id:        payId,
		TxId:      txId,
		IssueTxId: verifyResult.issueTxId,
		Packed:    packedTransfer,
		Payments:  payments,
	}

	// if already seen just return pay id and previous payments if present
	entry, ok := globalData.pendingTransactions[payId]
	if ok {
		if entry.payments != nil {
			result.Payments = entry.payments
		} else {
			// this would mean that reservoir data is corrupt
			logger.Panicf("storeTransfer: failed to get current payment data for: %s  payid: %s", txId, payId)
		}
		return result, true, nil
	}

	// if duplicates were detected, but different duplicates were present
	// then it is an error
	if duplicate {
		return nil, true, fault.TransactionAlreadyExists
	}

	transferredItem := &transactionData{
		txId:        txId,
		transaction: transfer,
		packed:      packedTransfer,
	}

	// already received the payment for the transfer
	// approve the transfer immediately if payment is ok
	detail, ok := globalData.orphanPayments[payId]
	if ok || globalData.autoVerify {
		if acceptablePayment(detail, payments) {
			globalData.verifiedTransactions[payId] = transferredItem
			globalData.verifiedIndex[txId] = payId
			globalData.inProgressLinks[transfer.GetLink()] = txId
			delete(globalData.pendingTransactions, payId)
			delete(globalData.pendingIndex, txId)
			delete(globalData.orphanPayments, payId)
			return result, false, nil
		}
	}

	// waiting for the payment to come
	payment := &transactionPaymentData{
		payId:     payId,
		tx:        transferredItem,
		payments:  payments,
		expiresAt: time.Now().Add(constants.ReservoirTimeout),
	}

	if len(globalData.pendingTransactions) >= maximumPendingTransactions {
		return nil, false, fault.BufferCapacityLimit
	}

	globalData.pendingTransactions[payId] = payment
	globalData.pendingIndex[txId] = payId
	globalData.inProgressLinks[transfer.GetLink()] = txId

	return result, false, nil
}

// verify that a transfer is ok
// ensure lock is held before calling
func verifyTransfer(transfer transactionrecord.BitmarkTransfer, transactionHandle storage.Handle, ownerTxHandle storage.Handle, ownerDataHandle storage.Handle) (*verifiedTransferInfo, bool, error) {

	// find the current owner via the link
	_, previousPacked := transactionHandle.GetNB(transfer.GetLink().Bytes())
	if previousPacked == nil {
		return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
	}

	previousTransaction, _, err := transactionrecord.Packed(previousPacked).Unpack(mode.IsTesting())
	if err != nil {
		return nil, false, err
	}

	var currentOwner *account.Account
	var previousTransfer transactionrecord.BitmarkTransfer

	// ensure that the transaction is a valid testChain transition
	switch tx := previousTransaction.(type) {
	case *transactionrecord.BitmarkIssue:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned, *transactionrecord.BitmarkShare:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BitmarkTransferUnratified:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned, *transactionrecord.BitmarkShare:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BitmarkTransferCountersigned:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned, *transactionrecord.BitmarkShare:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.OldBaseData:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BlockFoundation:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BlockOwnerTransfer:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
		}

	default:
		return nil, false, fault.LinkToInvalidOrUnconfirmedTransaction
	}

	// pack transfer and check signature
	packedTransfer, err := transfer.Pack(currentOwner)
	if err != nil {
		return nil, false, err
	}

	// transfer identifier and check for duplicate
	txId := packedTransfer.MakeLink()
	link := transfer.GetLink()
	if txId == link {
		// reject any transaction that links to itself
		// this should never occur, but protect against this situation
		return nil, false, fault.TransactionLinksToSelf
	}

	// check for double spend
	linkTxId, okL := globalData.inProgressLinks[link]
	_, okP := globalData.pendingIndex[txId]
	_, okV := globalData.verifiedIndex[txId]

	if okL && linkTxId != txId {
		// not an exact match - must be a double transfer
		return nil, false, fault.DoubleTransferAttempt
	}

	duplicate := false
	if okP {
		// if both then it is a possible duplicate
		// (depends on later pay id check)
		duplicate = true
	}

	// a single verified transfer fails the whole block
	if okV {
		return nil, false, fault.TransactionAlreadyExists
	}
	// a single confirmed transfer fails the whole block
	if transactionHandle.Has(txId[:]) {
		return nil, false, fault.TransactionAlreadyExists
	}

	// log.Infof("packed transfer: %x", packedTransfer)
	// log.Infof("id: %v", txId)

	// get count for current owner record
	// to make sure that the record has not already been transferred
	dKey := append(currentOwner.Bytes(), link[:]...)
	// log.Infof("dKey: %x", dKey)
	dCount := ownerTxHandle.Get(dKey)
	if dCount == nil {
		return nil, false, fault.DoubleTransferAttempt
	}

	// get ownership data
	ownerData, err := ownership.GetOwnerData(nil, link, ownerDataHandle)
	if err != nil {
		globalData.log.Errorf("owner data error: %s", err)
		return nil, false, err //fault.DoubleTransferAttempt
	}
	// log.Debugf("ownerData: %x", ownerData)

	result := &verifiedTransferInfo{
		txId:                txId,
		packed:              packedTransfer,
		previousTransfer:    previousTransfer,
		issueTxId:           ownerData.IssueTxId(),
		transferBlockNumber: ownerData.TransferBlockNumber(),
		issueBlockNumber:    ownerData.IssueBlockNumber(),
	}
	return result, duplicate, nil
}
