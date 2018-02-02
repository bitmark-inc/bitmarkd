// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// result returned by store transfer
type TransferInfo struct {
	Id       pay.PayId
	TxId     merkle.Digest
	Packed   []byte
	Payments []transactionrecord.PaymentAlternative
}

func StoreTransfer(transfer transactionrecord.BitmarkTransfer) (*TransferInfo, bool, error) {
	verifyResult, duplicate, err := verifyTransfer(transfer)
	if err != nil {
		return nil, false, err
	}

	// compute pay id
	packedTransfer := verifyResult.packedTransfer
	payId := pay.NewPayId([][]byte{packedTransfer})

	txId := verifyResult.txId
	link := transfer.GetLink()
	if txId == link {
		// reject any transaction that links to itself
		// this should never occur, but protect against this situuation
		return nil, false, fault.ErrTransactionLinksToSelf
	}

	previousTransfer := verifyResult.previousTransfer
	ownerData := verifyResult.ownerData

	payments := getPayments(ownerData, previousTransfer)

	result := &TransferInfo{
		Id:       payId,
		TxId:     txId,
		Packed:   packedTransfer,
		Payments: payments,
	}

	// if already seen just return pay id
	if _, ok := cache.Pool.UnverifiedTxEntries.Get(payId.String()); ok {
		return result, true, nil
	}

	// if duplicates were detected, but different duplicates were present
	// then it is an error
	if duplicate {
		return nil, true, fault.ErrTransactionAlreadyExists
	}

	transferredItem := &itemData{
		txIds:        []merkle.Digest{txId},
		links:        []merkle.Digest{link},
		transactions: [][]byte{packedTransfer},
	}

	// already received the payment for the transfer
	// approve the transfer immediately if payment is ok
	if val, ok := cache.Pool.OrphanPayment.Get(payId.String()); ok {
		detail := val.(*PaymentDetail)

		if acceptablePayment(detail, payments) {

			cache.Pool.VerifiedTx.Put(
				txId.String(),
				&verifiedItem{
					itemData:    transferredItem,
					transaction: packedTransfer,
					index:       0,
				},
			)
			cache.Pool.OrphanPayment.Delete(payId.String())
			return result, false, nil
		}
	}

	// waiting for the payment to come
	cache.Pool.PendingTransfer.Put(link.String(), txId)
	cache.Pool.UnverifiedTxIndex.Put(txId.String(), payId)
	cache.Pool.UnverifiedTxEntries.Put(
		payId.String(),
		&unverifiedItem{
			itemData: transferredItem,
			payments: payments,
		},
	)

	return result, false, nil
}

// returned data from verifyTransfer
type verifiedInfo struct {
	txId             merkle.Digest
	packedTransfer   []byte
	previousTransfer transactionrecord.BitmarkTransfer
	ownerData        []byte
}

// verify that a transfer is ok
// ensure lock is held before calling
func verifyTransfer(newTransfer transactionrecord.BitmarkTransfer) (*verifiedInfo, bool, error) {

	// find the current owner via the link
	previousPacked := storage.Pool.Transactions.Get(newTransfer.GetLink().Bytes())
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
		switch newTransfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BitmarkTransferUnratified:
		// ensure link to correct transfer type
		switch newTransfer.(type) {
		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BitmarkTransferCountersigned:
		// do not permit transfer downgrade
		switch newTransfer.(type) {
		case *transactionrecord.BitmarkTransferCountersigned:
			currentOwner = tx.Owner
			previousTransfer = tx
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.OldBaseData:
		// ensure link to correct transfer type
		switch newTransfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BlockFoundation:
		// ensure link to correct transfer type
		switch newTransfer.(type) {
		case *transactionrecord.BlockOwnerTransfer:
			currentOwner = tx.Owner
		default:
			return nil, false, fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

	case *transactionrecord.BlockOwnerTransfer:
		// ensure link to correct transfer type
		switch newTransfer.(type) {
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
	packedTransfer, err := newTransfer.Pack(currentOwner)
	if nil != err {
		return nil, false, err
	}

	// transfer identifier and check for duplicate
	txId := packedTransfer.MakeLink()

	// check if this transfer was already received
	_, okP := cache.Pool.PendingTransfer.Get(newTransfer.GetLink().String())
	_, okU := cache.Pool.UnverifiedTxIndex.Get(txId.String())
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
	if _, ok := cache.Pool.VerifiedTx.Get(txId.String()); ok {
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
	dKey := append(currentOwner.Bytes(), newTransfer.GetLink().Bytes()...)
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
