// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// result returned by store transfer
type TransferInfo struct {
	Id       pay.PayId
	TxId     merkle.Digest
	Packed   []byte
	Payments []transactionrecord.PaymentAlternative
}

// returned data from verifyTransfer
type verifiedTransferInfo struct {
	txId             merkle.Digest
	packed           []byte
	previousTransfer transactionrecord.BitmarkTransfer
	ownerData        []byte
}

func StoreTransfer(transfer transactionrecord.BitmarkTransfer) (*TransferInfo, bool, error) {
	verifyResult, duplicate, err := verifyTransfer(transfer)
	if err != nil {
		return nil, false, err
	}

	// compute pay id
	packedTransfer := verifyResult.packed
	payId := pay.NewPayId([][]byte{packedTransfer})

	txId := verifyResult.txId

	previousTransfer := verifyResult.previousTransfer
	ownerData := verifyResult.ownerData

	payments := getPayments(ownerData, previousTransfer)

	result := &TransferInfo{
		Id:       payId,
		TxId:     txId,
		Packed:   packedTransfer,
		Payments: payments,
	}

	// if already seen just return pay id and previous payments if present
	globalData.RLock()
	entry, ok := globalData.pendingTransactions[payId]
	globalData.RUnlock()
	if ok {
		if nil != entry.payments {
			result.Payments = entry.payments
		} else {
			// this would mean that reservoir data is corrupt
			logger.Panicf("StoreTransfer: failed to get current payment data for: %s  payid: %s", txId, payId)
		}
		return result, true, nil
	}

	// if duplicates were detected, but different duplicates were present
	// then it is an error
	if duplicate {
		return nil, true, fault.ErrTransactionAlreadyExists
	}

	transferredItem := &transactionData{
		txId:        txId,
		transaction: transfer,
		packed:      packedTransfer,
	}

	// already received the payment for the transfer
	// approve the transfer immediately if payment is ok
	globalData.RLock()
	detail, ok := globalData.orphanPayments[payId]
	globalData.RUnlock()
	if ok {
		if acceptablePayment(detail, payments) {
			globalData.Lock()
			globalData.verifiedTransactions[payId] = transferredItem
			globalData.inProgressLinks[transfer.GetLink()] = txId
			delete(globalData.pendingTransactions, payId)
			delete(globalData.pendingIndex, txId)
			delete(globalData.orphanPayments, payId)
			globalData.Unlock()
			return result, false, nil
		}
	}

	// waiting for the payment to come
	payment := &transactionPaymentData{
		payId:    payId,
		tx:       transferredItem,
		payments: payments,
	}

	globalData.Lock()

	if len(globalData.pendingTransactions) >= maximumPendingTransactions {
		globalData.Unlock()
		return nil, false, fault.ErrBufferCapacityLimit
	}

	globalData.pendingTransactions[payId] = payment
	globalData.pendingIndex[txId] = payId
	globalData.inProgressLinks[transfer.GetLink()] = txId
	globalData.Unlock()

	return result, false, nil
}

// verify that a transfer is ok
// ensure lock is held before calling
func verifyTransfer(transfer transactionrecord.BitmarkTransfer) (*verifiedTransferInfo, bool, error) {

	// find the current owner via the link
	_, previousPacked := storage.Pool.Transactions.GetNB(transfer.GetLink().Bytes())
	if nil == previousPacked {
		return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
	}

	previousTransaction, _, err := transactionrecord.Packed(previousPacked).Unpack(mode.IsTesting())
	if nil != err {
		return nil, false, err
	}

	var currentOwner *account.Account
	var previousTransfer transactionrecord.BitmarkTransfer

	// ensure that the transaction is a valid chain transition
	switch tx := previousTransaction.(type) {
	case *transactionrecord.BitmarkIssue:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BitmarkTransferUnratified:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BitmarkTransferCountersigned:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.OldBaseData:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BlockFoundation:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BlockOwnerTransfer:
		// ensure link to correct transfer type
		switch transfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	default:
		return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
	}

	// pack transfer and check signature
	packedTransfer, err := transfer.Pack(currentOwner)
	if nil != err {
		return nil, false, err
	}

	// transfer identifier and check for duplicate
	txId := packedTransfer.MakeLink()
	link := transfer.GetLink()
	if txId == link {
		// reject any transaction that links to itself
		// this should never occur, but protect against this situation
		return nil, false, fault.ErrTransactionLinksToSelf
	}

	// check for double spend
	globalData.RLock()
	linkTxId, okL := globalData.inProgressLinks[link]
	_, okP := globalData.pendingIndex[txId]
	_, okV := globalData.verifiedIndex[txId]
	globalData.RUnlock()

	if okL && linkTxId != txId {
		// not an exact match - must be a double transfer
		return nil, false, fault.ErrDoubleTransferAttempt
	}

	duplicate := false
	if okP {
		// if both then it is a possible duplicate
		// (depends on later pay id check)
		duplicate = true
	}

	// a single verified transfer fails the whole block
	if okV {
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
	dKey := append(currentOwner.Bytes(), transfer.GetLink().Bytes()...)
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

	result := &verifiedTransferInfo{
		txId:             txId,
		packed:           packedTransfer,
		previousTransfer: previousTransfer,
		ownerData:        ownerData,
	}
	return result, duplicate, nil
}
