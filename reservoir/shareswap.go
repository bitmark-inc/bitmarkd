// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// SwapInfo - result returned by store share
type SwapInfo struct {
	RemainingOne uint64
	RemainingTwo uint64
	Id           pay.PayId
	TxId         merkle.Digest
	Packed       []byte
	Payments     []transactionrecord.PaymentAlternative
}

// returned data from verifySwap
type verifiedSwapInfo struct {
	balanceOne          uint64
	balanceTwo          uint64
	txId                merkle.Digest
	packed              []byte
	issueTxId           merkle.Digest
	transferBlockNumber uint64
	issueBlockNumber    uint64
}

// StoreSwap - verify and store a swap request
func StoreSwap(swap *transactionrecord.ShareSwap) (*SwapInfo, bool, error) {

	globalData.Lock()
	defer globalData.Unlock()

	verifyResult, duplicate, err := verifySwap(swap)
	if err != nil {
		return nil, false, err
	}

	// compute pay id
	packedSwap := verifyResult.packed
	payId := pay.NewPayId([][]byte{packedSwap})

	txId := verifyResult.txId

	payments := getPayments(verifyResult.transferBlockNumber, verifyResult.issueBlockNumber, nil)

	spendKeyOne := makeSpendKey(swap.OwnerOne, swap.ShareIdOne)
	spendKeyTwo := makeSpendKey(swap.OwnerTwo, swap.ShareIdTwo)

	spendOne, ok := globalData.spend[spendKeyOne]
	spendTwo, ok := globalData.spend[spendKeyTwo]

	result := &SwapInfo{
		RemainingOne: verifyResult.balanceOne - spendOne,
		RemainingTwo: verifyResult.balanceTwo - spendTwo,
		Id:           payId,
		TxId:         txId,
		Packed:       packedSwap,
		Payments:     payments,
	}

	// if already seen just return pay id and previous payments if present
	entry, ok := globalData.pendingTransactions[payId]
	if ok {
		if nil != entry.payments {
			result.Payments = entry.payments
		} else {
			// this would mean that reservoir data is corrupt
			logger.Panicf("StoreSwap: failed to get current payment data for: %s  payid: %s", txId, payId)
		}
		return result, true, nil
	}

	// if duplicates were detected, but different duplicates were present
	// then it is an error
	if duplicate {
		return nil, true, fault.ErrTransactionAlreadyExists
	}

	swapItem := &transactionData{
		txId:        txId,
		transaction: swap,
		packed:      packedSwap,
	}

	// already received the payment for the swap
	// approve the swap immediately if payment is ok
	detail, ok := globalData.orphanPayments[payId]
	if ok {
		if acceptablePayment(detail, payments) {
			globalData.verifiedTransactions[payId] = swapItem
			globalData.verifiedIndex[txId] = payId
			delete(globalData.pendingTransactions, payId)
			delete(globalData.pendingIndex, txId)
			delete(globalData.orphanPayments, payId)

			globalData.spend[spendKeyOne] += swap.QuantityOne
			globalData.spend[spendKeyTwo] += swap.QuantityTwo
			result.RemainingOne -= swap.QuantityOne
			result.RemainingTwo -= swap.QuantityTwo
			return result, false, nil
		}
	}

	// waiting for the payment to come
	payment := &transactionPaymentData{
		payId:     payId,
		tx:        swapItem,
		payments:  payments,
		expiresAt: time.Now().Add(constants.ReservoirTimeout),
	}

	globalData.pendingTransactions[payId] = payment
	globalData.pendingIndex[txId] = payId
	globalData.spend[spendKeyOne] += swap.QuantityOne
	globalData.spend[spendKeyTwo] += swap.QuantityTwo
	result.RemainingOne -= swap.QuantityOne
	result.RemainingTwo -= swap.QuantityTwo

	return result, false, nil
}

// CheckSwapBalances - check sufficient balance on both accounts to be able to execute a swap request
func CheckSwapBalances(swap *transactionrecord.ShareSwap) (uint64, uint64, error) {

	// check incoming quantity
	if 0 == swap.QuantityOne || 0 == swap.QuantityTwo {
		return 0, 0, fault.ErrShareQuantityTooSmall
	}

	oKeyOne := append(swap.OwnerOne.Bytes(), swap.ShareIdOne[:]...)
	balanceOne, ok := storage.Pool.ShareQuantity.GetN(oKeyOne)

	// check if sufficient funds
	if !ok || balanceOne < swap.QuantityOne {
		return 0, 0, fault.ErrInsufficientShares
	}

	oKeyTwo := append(swap.OwnerTwo.Bytes(), swap.ShareIdTwo[:]...)
	balanceTwo, ok := storage.Pool.ShareQuantity.GetN(oKeyTwo)

	// check if sufficient funds
	if !ok || balanceTwo < swap.QuantityTwo {
		return 0, 0, fault.ErrInsufficientShares
	}

	return balanceOne, balanceTwo, nil
}

// verify that a swap is ok
// ensure lock is held before calling
func verifySwap(swap *transactionrecord.ShareSwap) (*verifiedSwapInfo, bool, error) {

	height := blockheader.Height()
	if swap.BeforeBlock <= height {
		return nil, false, fault.ErrRecordHasExpired
	}

	balanceOne, balanceTwo, err := CheckSwapBalances(swap)
	if nil != err {
		return nil, false, err
	}

	// pack swap and check signature
	packedSwap, err := swap.Pack(swap.OwnerOne)
	if nil != err {
		return nil, false, err
	}

	// transfer identifier and check for duplicate
	txId := packedSwap.MakeLink()

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

	// log.Infof("share one: %x", swap.ShareOne)
	// log.Infof("share two: %x", swap.ShareTwo)

	// strip off the total leaving just normal ownerdata layout
	// ***** FIX THIS: only Share One for owner data to determine payment?
	// ***** FIX THIS: should share two's owner dat be used for double charge?
	// the owner data is under tx id of share record
	_ /*totalValue*/, shareTxId := storage.Pool.Shares.GetNB(swap.ShareIdOne[:])
	if nil == shareTxId {
		return nil, false, fault.ErrDoubleTransferAttempt
	}
	ownerData, err := ownership.GetOwnerDataB(shareTxId)
	if nil != err {
		return nil, false, fault.ErrDoubleTransferAttempt
	}
	// log.Infof("ownerData: %x", ownerData)

	result := &verifiedSwapInfo{
		balanceOne:          balanceOne,
		balanceTwo:          balanceTwo,
		txId:                txId,
		packed:              packedSwap,
		issueTxId:           ownerData.IssueTxId(),
		transferBlockNumber: ownerData.TransferBlockNumber(),
		issueBlockNumber:    ownerData.IssueBlockNumber(),
	}
	return result, duplicate, nil
}
