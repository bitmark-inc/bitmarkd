// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// result returned by store share
type GrantInfo struct {
	Remaining uint64
	Id        pay.PayId
	TxId      merkle.Digest
	Packed    []byte
	Payments  []transactionrecord.PaymentAlternative
}

// returned data from verifyGrant
type verifiedGrantInfo struct {
	balance             uint64
	txId                merkle.Digest
	packed              []byte
	issueTxId           merkle.Digest
	transferBlockNumber uint64
	issueBlockNumber    uint64
}

func StoreGrant(grant *transactionrecord.ShareGrant) (*GrantInfo, bool, error) {

	globalData.Lock()
	defer globalData.Unlock()

	verifyResult, duplicate, err := verifyGrant(grant)
	if err != nil {
		return nil, false, err
	}

	// compute pay id
	packedGrant := verifyResult.packed
	payId := pay.NewPayId([][]byte{packedGrant})

	txId := verifyResult.txId

	payments := getPayments(verifyResult.transferBlockNumber, verifyResult.issueBlockNumber, nil)

	spendKey := makeSpendKey(grant.Owner, grant.ShareId)

	spend, ok := globalData.spend[spendKey]

	result := &GrantInfo{
		Remaining: verifyResult.balance - spend,
		Id:        payId,
		TxId:      txId,
		Packed:    packedGrant,
		Payments:  payments,
	}

	// if already seen just return pay id and previous payments if present
	entry, ok := globalData.pendingTransactions[payId]
	if ok {
		if nil != entry.payments {
			result.Payments = entry.payments
		} else {
			// this would mean that reservoir data is corrupt
			logger.Panicf("StoreGrant: failed to get current payment data for: %s  payid: %s", txId, payId)
		}
		return result, true, nil
	}

	// if duplicates were detected, but different duplicates were present
	// then it is an error
	if duplicate {
		return nil, true, fault.ErrTransactionAlreadyExists
	}

	grantItem := &transactionData{
		txId:        txId,
		transaction: grant,
		packed:      packedGrant,
	}

	// already received the payment for the grant
	// approve the grant immediately if payment is ok
	detail, ok := globalData.orphanPayments[payId]
	if ok {
		if acceptablePayment(detail, payments) {
			globalData.verifiedTransactions[payId] = grantItem
			globalData.verifiedIndex[txId] = payId
			delete(globalData.pendingTransactions, payId)
			delete(globalData.pendingIndex, txId)
			delete(globalData.orphanPayments, payId)

			globalData.spend[spendKey] += grant.Quantity
			result.Remaining -= grant.Quantity
			return result, false, nil
		}
	}

	// waiting for the payment to come
	payment := &transactionPaymentData{
		payId:    payId,
		tx:       grantItem,
		payments: payments,
	}

	globalData.pendingTransactions[payId] = payment
	globalData.pendingIndex[txId] = payId
	globalData.spend[spendKey] += grant.Quantity
	result.Remaining -= grant.Quantity

	return result, false, nil
}

func makeSpendKey(owner *account.Account, shareId merkle.Digest) spendKey {

	oKey := spendKey{
		share: shareId,
	}

	ob := owner.Bytes()
	if len(ob) > len(oKey.owner) {
		logger.Panicf("StoreGrant: owner bytes length: %d expected less than: %d", len(ob), len(oKey.owner))
	}
	copy(oKey.owner[:], ob)
	return oKey
}

func CheckGrantBalance(grant *transactionrecord.ShareGrant) (uint64, error) {

	// check incoming quantity
	if 0 == grant.Quantity {
		return 0, fault.ErrShareQuantityTooSmall
	}

	oKey := append(grant.Owner.Bytes(), grant.ShareId[:]...)
	balance, ok := storage.Pool.ShareQuantity.GetN(oKey)

	// check if sufficient funds
	if !ok || balance < grant.Quantity {
		return 0, fault.ErrInsufficientShares
	}

	return balance, nil
}

// verify that a grant is ok
func verifyGrant(grant *transactionrecord.ShareGrant) (*verifiedGrantInfo, bool, error) {

	height := blockheader.Height()
	if grant.BeforeBlock <= height {
		return nil, false, fault.ErrRecordHasExpired
	}

	balance, err := CheckGrantBalance(grant)
	if nil != err {
		return nil, false, err
	}

	// pack grant and check signature
	packedGrant, err := grant.Pack(grant.Owner)
	if nil != err {
		return nil, false, err
	}

	// transfer identifier and check for duplicate
	txId := packedGrant.MakeLink()

	// check for double spend
	_, okP := globalData.pendingIndex[txId]
	_, okV := globalData.verifiedIndex[txId]

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

	// log.Infof("share: %x", grant.Share)

	// the owner data is under tx id of share record
	_ /*totalValue*/, shareTxId := storage.Pool.Shares.GetNB(grant.ShareId[:])
	if nil == shareTxId {
		return nil, false, fault.ErrDoubleTransferAttempt
	}

	ownerData, err := ownership.GetOwnerDataB(shareTxId)
	if nil != err {
		return nil, false, fault.ErrDoubleTransferAttempt
	}
	// log.Debugf("ownerData: %x", ownerData)

	result := &verifiedGrantInfo{
		balance:             balance,
		txId:                txId,
		packed:              packedGrant,
		issueTxId:           ownerData.IssueTxId(),
		transferBlockNumber: ownerData.TransferBlockNumber(),
		issueBlockNumber:    ownerData.IssueBlockNumber(),
	}
	return result, duplicate, nil
}
