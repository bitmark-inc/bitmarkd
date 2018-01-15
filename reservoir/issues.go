// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"golang.org/x/crypto/sha3"
	"math/big"
)

const (
	maximumIssues = 100 // allowed issues in a single submission
)

// result returned by store issues
type IssueInfo struct {
	Id         pay.PayId
	Nonce      PayNonce
	Difficulty *difficulty.Difficulty
	TxIds      []merkle.Digest
	Packed     []byte
}

// store packed record(s) in the Unverified table
//
// return payment id and a duplicate flag
//
// for duplicate to be true all transactions must all match exactly to a
// previous set - this is to allow for multiple submission from client
// without receiving a duplicate transaction error
func StoreIssues(issues []*transactionrecord.BitmarkIssue, isVerified bool) (*IssueInfo, bool, error) {

	count := len(issues)
	if count > maximumIssues {
		return nil, false, fault.ErrTooManyItemsToProcess
	} else if 0 == count {
		return nil, false, fault.ErrMissingParameters
	}

	// individual packed issues
	separated := make([][]byte, count)

	// all the tx id corresponding to separated
	txIds := make([]merkle.Digest, count)
	// all the assets id corresponding to separated
	assetIds := make([]transactionrecord.AssetIndex, count)

	// this flags already stored issues
	// used to flag an error if pay id is different
	// as this would be an overlapping block of issues
	duplicate := false

	// verify each transaction
	for i, issue := range issues {

		if issue.Owner.IsTesting() != mode.IsTesting() {
			return nil, false, fault.ErrWrongNetworkForPublicKey
		}

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
		if _, ok := cache.Pool.UnverifiedTxIndex.Get(txId.String()); ok {
			// if duplicate, activate pay id check
			duplicate = true
		}

		// a single verified issue fails the whole block
		if _, ok := cache.Pool.VerifiedTx.Get(txId.String()); ok {
			return nil, false, fault.ErrTransactionAlreadyExists
		}
		// a single confirmed issue fails the whole block
		if storage.Pool.Transactions.Has(txId[:]) {
			return nil, false, fault.ErrTransactionAlreadyExists
		}

		// accumulate the data
		txIds[i] = txId
		assetIds[i] = issue.AssetIndex
		separated[i] = packedIssue

		// this length of the verified issues should be exactly one
		// the verified issue will be stored directly
		if len(issues) == 1 && !duplicate && isVerified {
			transactions := [][]byte{packedIssue[:]}

			v := &verifiedItem{
				itemData: &itemData{
					txIds:        txIds,
					links:        nil,
					assetIds:     assetIds,
					transactions: transactions,
				},
				transaction: packedIssue,
			}
			cache.Pool.VerifiedTx.Put(txId.String(), v)
			return nil, false, nil
		}
	}

	// compute pay id
	payId := pay.NewPayId(separated)
	nonce := NewPayNonce()
	difficulty := ScaledDifficulty(count)

	result := &IssueInfo{
		Id:         payId,
		Nonce:      nonce,
		Difficulty: difficulty,
		TxIds:      txIds,
		Packed:     bytes.Join(separated, []byte{}),
	}

	// if already seen just return pay id
	if _, ok := cache.Pool.UnverifiedTxEntries.Get(payId.String()); ok {
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

	// create index entries
	for _, txId := range txIds {
		cache.Pool.UnverifiedTxIndex.Put(txId.String(), payId)
	}

	// save transactions
	entry := &unverifiedItem{
		itemData: &itemData{
			txIds:        txIds,
			links:        nil,
			assetIds:     assetIds,
			transactions: separated,
		},
		nonce:      nonce, // ***** FIX THIS: this value seems not used
		difficulty: difficulty,
	}
	//copy(entry.txIds, txIds)
	//copy(entry.transactions, transactions)

	cache.Pool.UnverifiedTxEntries.Put(payId.String(), entry)

	return result, false, nil
}

// instead of paying, try a proof from the client nonce
func TryProof(payId pay.PayId, clientNonce []byte) TrackingStatus {
	val, ok := cache.Pool.UnverifiedTxEntries.Get(payId.String())

	//	r, done, ok := get(payId)
	if !ok {
		globalData.log.Debugf("TryProof: issue item not found")
		return TrackingNotFound
	}

	r := val.(*unverifiedItem)
	// if done {
	// 	return TrackingProcessed
	// }
	if nil == r.difficulty { // only payment tracking; proof not allowed
		globalData.log.Debugf("TryProof: item with out a difficulty")
		return TrackingInvalid
	}

	// convert difficulty
	bigDifficulty := r.difficulty.BigInt()

	globalData.log.Infof("TryProof: difficulty: 0x%064x", bigDifficulty)

	var payNonce [8]byte
	// compute hash with all possible payNonces
	h := sha3.New256()
	iterator := blockring.NewRingReader()
	i := 0 // ***** FIX THIS: debug
	for iterator.Next() {
		crc := iterator.GetCRC()
		binary.BigEndian.PutUint64(payNonce[:], crc)
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
			setVerified(payId, nil)
			return TrackingAccepted
		}
	}
	return TrackingInvalid
}
