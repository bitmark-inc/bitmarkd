// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// result returned by store issues
type IssueInfo struct {
	TxIds      []merkle.Digest
	Packed     []byte
	Id         pay.PayId
	Nonce      PayNonce
	Difficulty *difficulty.Difficulty
	Payments   []transactionrecord.PaymentAlternative
}

// store packed record(s) in the pending table
//
// return payment id and a duplicate flag
//
// for duplicate to be true all transactions must all match exactly to a
// previous set - this is to allow for multiple submission from client
// without receiving a duplicate transaction error
func StoreIssues(issues []*transactionrecord.BitmarkIssue) (*IssueInfo, bool, error) {

	count := len(issues)
	if count > MaximumIssues {
		return nil, false, fault.ErrTooManyItemsToProcess
	} else if 0 == count {
		return nil, false, fault.ErrMissingParameters
	}

	// individual packed issues
	separated := make([][]byte, count)

	// all the tx id corresponding to separated
	txIds := make([]merkle.Digest, count)

	// check if different assets
	uniqueAssetId := issues[0].AssetId
	unique := true

	// this flags already stored issues
	// used to flag an error if pay id is different
	// as this would be an overlapping block of issues
	duplicate := false

	// only allow free issues if all nonces are zero
	freeIssueAllowed := true

	// verify each transaction
	for i, issue := range issues {

		if nil == issue || nil == issue.Owner {
			return nil, false, fault.ErrInvalidItem
		}

		if issue.Owner.IsTesting() != mode.IsTesting() {
			return nil, false, fault.ErrWrongNetworkForPublicKey
		}

		// all are free or all are non-free
		if 0 != issue.Nonce {
			freeIssueAllowed = false
		}

		// validate issue record
		packedIssue, err := issue.Pack(issue.Owner)
		if nil != err {
			return nil, false, err
		}

		if !asset.Exists(issue.AssetId) {
			return nil, false, fault.ErrAssetNotFound
		}

		txId := packedIssue.MakeLink()

		// an unverified issue tag the block as possible duplicate
		// (if pay id matched later)
		globalData.RLock()
		_, ok := globalData.pendingIndex[txId]
		if ok {
			// if duplicate, activate pay id check
			duplicate = true
		}

		// a single verified issue fails the whole block
		_, ok = globalData.verifiedIndex[txId]
		globalData.RUnlock()
		if ok {
			return nil, false, fault.ErrTransactionAlreadyExists
		}
		// a single confirmed issue fails the whole block
		if storage.Pool.Transactions.Has(txId[:]) {
			return nil, false, fault.ErrTransactionAlreadyExists
		}

		// accumulate the data
		txIds[i] = txId
		if uniqueAssetId != issue.AssetId {
			unique = false
		}
		separated[i] = packedIssue
	}

	// compute pay id
	payId := pay.NewPayId(separated)

	// compose new result
	result := &IssueInfo{
		Id:     payId,
		TxIds:  txIds,
		Packed: bytes.Join(separated, []byte{}),
		//Nonce:      nil,
		Difficulty: nil,
		Payments:   nil,
	}

	// check if already seen
	globalData.RLock()
	if entry, ok := globalData.pendingFreeIssues[payId]; ok {

		globalData.log.Debugf("duplicate free issue pay id: %s", payId)

		result.Nonce = entry.nonce
		result.Difficulty = entry.difficulty

		globalData.RUnlock()

		return result, true, nil
	}

	if entry, ok := globalData.pendingPaidIssues[payId]; ok {

		globalData.log.Debugf("duplicate free issue pay id: %s", payId)

		result.Payments = entry.payments
		globalData.RUnlock()

		return result, true, nil
	}
	globalData.RUnlock()

	// if duplicates were detected, but duplicates were present
	// then it is an error
	if duplicate {
		globalData.log.Debugf("overlapping pay id: %s", payId)
		return nil, false, fault.ErrTransactionAlreadyExists
	}

	globalData.log.Infof("creating pay id: %s", payId)

	if freeIssueAllowed {
		result.Nonce = NewPayNonce()
		result.Difficulty = ScaledDifficulty(count)

	} else {
		// check for single asset being issued (paid issues)
		// fail if not a single confirmed asset
		if !unique {
			return nil, false, fault.ErrAssetNotFound
		}

		assetBlockNumber, t := storage.Pool.Assets.GetNB(uniqueAssetId[:])

		if nil == t || assetBlockNumber <= genesis.BlockNumber {
			return nil, false, fault.ErrAssetNotFound
		}

		blockNumberKey := make([]byte, 8)
		binary.BigEndian.PutUint64(blockNumberKey, assetBlockNumber)

		p := getPayment(blockNumberKey)
		if nil == p { // would be an internal database error
			globalData.log.Errorf("missing payment for asset id: %s", issues[0].AssetId)
			return nil, false, fault.ErrAssetNotFound
		}

		result.Payments = make([]transactionrecord.PaymentAlternative, 0, len(p))
		// multiply fees for each currency
		for _, r := range p {
			total := r.Amount * uint64(len(txIds))
			pa := transactionrecord.PaymentAlternative{
				&transactionrecord.Payment{
					Currency: r.Currency,
					Address:  r.Address,
					Amount:   total,
				},
			}
			result.Payments = append(result.Payments, pa)
		}
	}

	// save transactions
	txs := make([]*transactionData, len(txIds))
	for i, txId := range txIds {
		txs[i] = &transactionData{
			txId:        txId,
			transaction: issues[i],
			packed:      separated[i],
		}
	}

	entry := &issuePaymentData{
		payId:     payId,
		txs:       txs,
		payments:  result.Payments,
		expiresAt: time.Now().Add(constants.ReservoirTimeout),
	}

	// code below modifies maps
	globalData.Lock()
	defer globalData.Unlock()

	// already received the payment for the issues
	// approve the transfer immediately if payment is ok
	detail, ok := globalData.orphanPayments[payId]
	if ok {
		if acceptablePayment(detail, result.Payments) {
			for _, txId := range txIds {
				globalData.verifiedIndex[txId] = payId
				delete(globalData.pendingIndex, txId)
			}
			globalData.verifiedPaidIssues[payId] = entry
			//delete(globalData.pendingPaidIssues, payId) // not created
			delete(globalData.orphanPayments, payId)
			return result, false, nil
		}
	}

	if freeIssueAllowed && globalData.pendingFreeCount+len(txs) > maximumPendingFreeIssues ||
		!freeIssueAllowed && globalData.pendingPaidCount+len(txs) >= maximumPendingPaidIssues {
		return nil, false, fault.ErrBufferCapacityLimit
	}

	// create index entries
	for _, txId := range txIds {
		globalData.pendingIndex[txId] = payId
	}

	if freeIssueAllowed {
		globalData.pendingFreeIssues[payId] = &issueFreeData{
			payId:      payId,
			txs:        txs,
			nonce:      result.Nonce,
			difficulty: result.Difficulty,
			expiresAt:  time.Now().Add(constants.ReservoirTimeout),
		}
		globalData.pendingFreeCount += len(txs)

	} else {
		globalData.pendingPaidIssues[payId] = entry
		globalData.pendingPaidCount += len(txs)
	}

	return result, false, nil
}

// instead of paying, try a proof from the client nonce
func TryProof(payId pay.PayId, clientNonce []byte) TrackingStatus {

	globalData.RLock()
	r, ok := globalData.pendingFreeIssues[payId]
	globalData.RUnlock()

	if !ok {
		globalData.log.Debugf("TryProof: issue item not found")
		return TrackingNotFound
	}

	if nil == r.difficulty { // only payment tracking; proof not allowed
		globalData.log.Debugf("TryProof: item with out a difficulty")
		return TrackingInvalid
	}

	// convert difficulty
	bigDifficulty := r.difficulty.BigInt()

	globalData.log.Infof("TryProof: difficulty: 0x%064x", bigDifficulty)

	// compute hash with all possible payNonces
	h := sha3.New256()

	height := blockheader.Height()
	for i := uint64(0); i < 20; i += 1 {

		payNonce := PayNonceFromBlock(height - i)

		globalData.log.Debugf("TryProof: payNonce[%d]: %x", i, payNonce)

		i += 1 // ***** FIX THIS: debug
		h.Reset()
		h.Write(payId[:])
		h.Write(payNonce[:])
		h.Write(clientNonce)
		var digest [32]byte
		h.Sum(digest[:0])

		//globalData.log.Debugf("TryProof: digest: %x", digest)

		// convert to big integer from BE byte slice
		bigDigest := new(big.Int).SetBytes(digest[:])

		globalData.log.Debugf("TryProof: digest: 0x%064x", bigDigest)

		// check difficulty and verify if ok
		if bigDigest.Cmp(bigDifficulty) <= 0 {
			globalData.log.Debugf("TryProof: success: pay id: %s", payId)
			verifyIssueByNonce(payId, clientNonce)
			return TrackingAccepted
		}
	}

	return TrackingInvalid
}

// move transaction(s) to verified cache
func verifyIssueByNonce(payId pay.PayId, nonce []byte) bool {

	if nil == nonce || 0 == len(nonce) {
		globalData.log.Warn("nonce nil or empty")
		return false
	}
	globalData.log.Infof("nonce: %x", nonce)

	globalData.Lock()
	defer globalData.Unlock()

	entry, ok := globalData.pendingFreeIssues[payId]
	if ok {

		copy(entry.nonce[:], nonce[:])

		// move each transaction to verified pool
		for _, tx := range entry.txs {
			delete(globalData.pendingIndex, tx.txId)
			globalData.verifiedIndex[tx.txId] = payId
		}

		// remove the pending data
		globalData.pendingFreeCount -= len(entry.txs)
		delete(globalData.pendingFreeIssues, payId)

		// add to verified
		globalData.verifiedFreeIssues[payId] = entry
	}

	return ok
}
