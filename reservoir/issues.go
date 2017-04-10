// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"golang.org/x/crypto/sha3"
	"math/big"
	"time"
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

	// critical code - prevent overlapping blocks of issues
	globalData.Lock()
	defer globalData.Unlock()

	// individual packed issues
	separated := make([][]byte, count)

	// all the tx id corresponding to separated
	txIds := make([]merkle.Digest, count)
	// all the assets id corresponding to separated
	assetIds := make([][]byte, count)

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
		assetIds[i] = issue.AssetIndex[:]
		separated[i] = packedIssue

		// this length of the verified issues should be exactly one
		// the verified issue will be stored directly
		if len(issues) == 1 && !duplicate && isVerified {
			transactions := [][]byte{packedIssue[:]}

			v := &verifiedItem{
				data: &itemData{
					txIds:        txIds,
					links:        nil,
					assetIds:     assetIds,
					transactions: transactions,
				},
				transaction: packedIssue,
			}
			globalData.verified[txId] = v
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

	expiresAt := time.Now().Add(constants.ReservoirTimeout)

	// create index entries
	for _, txId := range txIds {
		globalData.unverified.index[txId] = payId
	}

	// save transactions
	entry := &unverifiedItem{
		itemData: &itemData{
			txIds:        txIds,
			links:        nil,
			assetIds:     assetIds,
			transactions: separated,
		},
		nonce:      nonce, // FIXME: this value seems not used
		difficulty: difficulty,
		expires:    expiresAt,
	}
	//copy(entry.txIds, txIds)
	//copy(entry.transactions, transactions)

	globalData.unverified.entries[payId] = entry

	return result, false, nil
}

// instead of paying, try a proof from the client nonce
func TryProof(payId pay.PayId, clientNonce []byte) TrackingStatus {

	globalData.RLock()
	r, ok := globalData.unverified.entries[payId]
	globalData.RUnlock()
	//	r, done, ok := get(payId)
	if !ok {
		return TrackingNotFound
	}
	// if done {
	// 	return TrackingProcessed
	// }
	if nil == r.difficulty { // only payment tracking; proof not allowed
		return TrackingInvalid
	}

	// convert difficulty
	bigDifficulty := r.difficulty.BigInt()

	globalData.log.Infof("TryProof: difficulty: 0x%064x", bigDifficulty)

	// compute hash with all possible payNonces
	h := sha3.New256()
	payNonce := make([]byte, 8)
	iterator := blockring.NewRingReader()
	i := 0 // ***** FIX THIS: debug
	for crc, ok := iterator.Get(); ok; crc, ok = iterator.Get() {

		binary.BigEndian.PutUint64(payNonce[:], crc)
		i += 1 // ***** FIX THIS: debug
		globalData.log.Debugf("TryProof: payNonce[%d]: %x", i, payNonce)

		h.Reset()
		h.Write(payId[:])
		h.Write(payNonce)
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
			globalData.Lock()
			setVerified(payId)
			globalData.Unlock()
			return TrackingAccepted
		}
	}
	return TrackingInvalid
}
