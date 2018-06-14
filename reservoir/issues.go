// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/cache"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
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
	Payments   []transactionrecord.PaymentAlternative
}

// store packed record(s) in the Unverified table
//
// return payment id and a duplicate flag
//
// for duplicate to be true all transactions must all match exactly to a
// previous set - this is to allow for multiple submission from client
// without receiving a duplicate transaction error
func StoreIssues(issues []*transactionrecord.BitmarkIssue) (*IssueInfo, bool, error) {

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

	// deduplicated list of assets
	uniqueAssetIds := make(map[transactionrecord.AssetIdentifier]struct{})

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

		if !asset.Exists(issue.AssetId) {
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
		uniqueAssetIds[issue.AssetId] = struct{}{}
		separated[i] = packedIssue
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
		Payments:   nil,
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

	// check for single asset being issued
	assetBlockNumber := uint64(0)
scan_for_one_asset:
	for _, issue := range issues {
		bn, t := storage.Pool.Assets.GetNB(issue.AssetId[:])
		if nil == t || 0 == bn {
			assetBlockNumber = 0     // cannot determine a single payment block
			break scan_for_one_asset // because of unconfirmed asset
		} else if 0 == assetBlockNumber {
			assetBlockNumber = bn // block number of asset
		} else if assetBlockNumber != bn {
			assetBlockNumber = 0     // cannot determin a single payment block
			break scan_for_one_asset // because of multiple assets
		}
	}

	if assetBlockNumber > genesis.BlockNumber { // avoid genesis block

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
	entry := &unverifiedItem{
		itemData: &itemData{
			txIds:        txIds,
			links:        nil,
			assetIds:     uniqueAssetIds,
			transactions: separated,
			nonce:        nil,
		},
		//nonce:      nonce, // ***** FIX THIS: this value seems not used
		difficulty: difficulty,
		payments:   result.Payments,
	}

	// already received the payment for the issues
	// approve the transfer immediately if payment is ok
	if val, ok := cache.Pool.OrphanPayment.Get(payId.String()); ok {
		detail := val.(*PaymentDetail)

		if acceptablePayment(detail, result.Payments) {

			for i, txId := range txIds {
				cache.Pool.VerifiedTx.Put(
					txId.String(),
					&verifiedItem{
						itemData:    entry.itemData,
						transaction: separated[i],
						index:       i,
					},
				)
			}
			cache.Pool.OrphanPayment.Delete(payId.String())
			return result, false, nil
		}
	}

	// create index entries
	for _, txId := range txIds {
		cache.Pool.UnverifiedTxIndex.Put(txId.String(), payId)
	}
	cache.Pool.UnverifiedTxEntries.Put(payId.String(), entry)

	return result, false, nil
}

// instead of paying, try a proof from the client nonce
func TryProof(payId pay.PayId, clientNonce []byte) TrackingStatus {
	val, ok := cache.Pool.UnverifiedTxEntries.Get(payId.String())

	if !ok {
		globalData.log.Debugf("TryProof: issue item not found")
		return TrackingNotFound
	}

	r := val.(*unverifiedItem)

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
			setVerified(payId, nil, clientNonce)
			return TrackingAccepted
		}
	}
	return TrackingInvalid
}
