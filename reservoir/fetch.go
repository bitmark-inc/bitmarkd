// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

const (
	maximumFetchFreeIssues   = 2000 // one asset + one issue
	maximumFetchPaidIssues   = 2000 // issues only
	minimumFetchTransactions = 2000 // other transactions
)

// free issues

type freeIssueIter struct {
	data    <-chan *issueFreeData
	control chan<- struct{}
	run     bool
}

func newFreeIssueIter() *freeIssueIter {

	data := make(chan *issueFreeData)
	control := make(chan struct{})

	iter := &freeIssueIter{
		data:    data,
		control: control,
		run:     true,
	}

	go func(data chan<- *issueFreeData, control <-chan struct{}) {
	loop:
		for _, issue := range globalData.verifiedFreeIssues {
			if _, ok := <-control; !ok {
				break loop
			}
			data <- issue
		}
		<-control
		iter.run = false
		close(data)
	}(data, control)

	return iter
}

func (iter *freeIssueIter) Get() (*issueFreeData, bool) {
	if iter.run {
		iter.control <- struct{}{}
		item, ok := <-iter.data
		return item, ok
	}
	return nil, false
}

func (iter *freeIssueIter) Close() {
	close(iter.control)
}

// paid issues

type paidIssueIter struct {
	data    <-chan *issuePaymentData
	control chan<- struct{}
	run     bool
}

func newPaidIssueIter() *paidIssueIter {

	data := make(chan *issuePaymentData)
	control := make(chan struct{})

	iter := &paidIssueIter{
		data:    data,
		control: control,
		run:     true,
	}

	go func(data chan<- *issuePaymentData, control <-chan struct{}) {
	loop:
		for _, issue := range globalData.verifiedPaidIssues {
			if _, ok := <-control; !ok {
				break loop
			}
			data <- issue
		}
		<-control
		iter.run = false
		close(data)
	}(data, control)

	return iter
}

func (iter *paidIssueIter) Get() (*issuePaymentData, bool) {
	if iter.run {
		iter.control <- struct{}{}
		item, ok := <-iter.data
		return item, ok
	}
	return nil, false
}

func (iter *paidIssueIter) Close() {
	close(iter.control)
}

// transactions

type transactionIter struct {
	data    <-chan *transactionData
	control chan<- struct{}
	run     bool
}

func newTransactionIter() *transactionIter {

	data := make(chan *transactionData)
	control := make(chan struct{})

	iter := &transactionIter{
		data:    data,
		control: control,
		run:     true,
	}

	go func(data chan<- *transactionData, control <-chan struct{}) {
	loop:
		for _, issue := range globalData.verifiedTransactions {
			if _, ok := <-control; !ok {
				break loop
			}
			data <- issue
		}
		<-control
		iter.run = false
		close(data)
	}(data, control)

	return iter
}

func (iter *transactionIter) Get() (*transactionData, bool) {
	if iter.run {
		iter.control <- struct{}{}
		item, ok := <-iter.data
		return item, ok
	}
	return nil, false
}

func (iter *transactionIter) Close() {
	close(iter.control)
}

// FetchVerified - fetch a series of verified transactions
func FetchVerified(count int) ([]merkle.Digest, []byte, error) {
	if count <= 0 {
		return nil, nil, fault.InvalidCount
	}
	if count < minimumFetchTransactions {
		return nil, nil, fault.InvalidCount
	}

	// data collection to return
	txIds := make([]merkle.Digest, 0, count)
	txData := make([]byte, 0, 200*count) // some arbitrary start
	n := 0
	seenAsset := make(map[transactionrecord.AssetIdentifier]struct{})

	// append a record to the collection
	store := func(txId merkle.Digest, packed transactionrecord.Packed) {
		if count <= 0 {
			globalData.log.Critical("buffer overrun")
			logger.Panic("fetch verified: buffer overrun")
		}
		txData = append(txData, packed...)
		txIds = append(txIds, txId)
		n += 1
		count -= 1
	}

	// store a block of free issues
	storeFree := func(issue *issueFreeData) {
		for _, item := range issue.txs {
			tx, ok := item.transaction.(*transactionrecord.BitmarkIssue)
			if !ok {
				globalData.log.Criticalf("not an issue: %+v", tx)
				logger.Panicf("fetch verified: not an issue: %+v", tx)
			}

			if _, ok := seenAsset[tx.AssetId]; !ok {
				if !storage.Pool.Assets.Has(tx.AssetId[:]) {
					packedAsset := asset.Get(tx.AssetId)
					if packedAsset == nil {
						globalData.log.Criticalf("missing asset: %v", tx.AssetId)
						logger.Panicf("fetch verified missing asset: %v", tx.AssetId)
					}
					store(merkle.NewDigest(packedAsset), packedAsset)
					seenAsset[tx.AssetId] = struct{}{}
				}
			}
			store(item.txId, item.packed)
		}
	}

	// store a block of paid issues
	storePaid := func(issue *issuePaymentData) {
		for _, item := range issue.txs {
			tx, ok := item.transaction.(*transactionrecord.BitmarkIssue)
			if !ok {
				globalData.log.Criticalf("not an issue: %+v", tx)
				logger.Panicf("fetch verified: not an issue: %+v", tx)
			}
			if _, ok := seenAsset[tx.AssetId]; !ok {
				if !storage.Pool.Assets.Has(tx.AssetId[:]) {
					globalData.log.Criticalf("missing confirmed asset: %v", tx.AssetId)
					logger.Panicf("fetch verified: missing confirmed asset: %v", tx.AssetId)
				}
			}
			store(item.txId, item.packed)
		}
	}

	globalData.RLock()
	defer globalData.RUnlock()
	if globalData.enabled {

		//----------------------------------------------------------------------------------------
		log := globalData.log
		log.Infof("verifiedTransactions: %d", len(globalData.verifiedTransactions))
		log.Infof("verifiedFreeIssues: %d", len(globalData.verifiedFreeIssues))
		log.Infof("verifiedPaidIssues: %d", len(globalData.verifiedPaidIssues))
		log.Infof("verifiedIndex: %d", len(globalData.verifiedIndex))
		log.Infof("inProgressLinks: %d", len(globalData.inProgressLinks))
		log.Infof("pendingTransactions: %d", len(globalData.pendingTransactions))
		log.Infof("pendingFreeIssues: %d", len(globalData.pendingFreeIssues))
		log.Infof("pendingPaidIssues: %d", len(globalData.pendingPaidIssues))
		log.Infof("pendingIndex: %d", len(globalData.pendingIndex))

		log.Infof("pendingFreeCount: %d", globalData.pendingFreeCount)
		log.Infof("pendingPaidCount: %d", globalData.pendingPaidCount)

		log.Infof("orphanPayments: %d", len(globalData.orphanPayments))

		//----------------------------------------------------------------------------------------

		freeIter := newFreeIssueIter()
		defer freeIter.Close()
		paidIter := newPaidIssueIter()
		defer paidIter.Close()
		transactionIter := newTransactionIter()
		defer transactionIter.Close()

		// output some free issues attaching the asset record before them
		n = 0 // reset stored counter
	free_issues:
		for {
			if n >= maximumFetchFreeIssues {
				break free_issues
			}
			issue, ok := freeIter.Get()
			if !ok {
				break free_issues
			}
			storeFree(issue)
		}

		// output some paid issues (asset must be confirmed)
		n = 0 // reset stored counter
	paid_issues:
		for {
			issue, ok := paidIter.Get()
			if !ok {
				break paid_issues
			}
			if n >= maximumFetchPaidIssues {
				break paid_issues
			}
			storePaid(issue)
		}

		// fill remainder with transactions
	normal_transactions:
		for {
			tx, ok := transactionIter.Get()
			if !ok {
				break normal_transactions
			}

			store(tx.txId, tx.packed)

			if count <= 0 {
				break normal_transactions
			}
		}

		if count > 0 {
			// pack more issues
			n = 0
			freeFlag := true
			paidFlag := true

			for (freeFlag || paidFlag) && count > 0 {

				if freeFlag {
					issueFree, ok := freeIter.Get()
					if ok && len(issueFree.txs)*2 <= count {
						storeFree(issueFree)
					} else {
						freeFlag = false
					}
				}

				if paidFlag {
					issuePaid, ok := paidIter.Get()
					if ok && len(issuePaid.txs)*2 <= count {
						storePaid(issuePaid)
					} else {
						paidFlag = false
					}
				}
			}

		}
	}

	globalData.log.Infof("tx ids: %v", txIds)
	globalData.log.Infof("tx data: %x", txData)

	return txIds, txData, nil
}
