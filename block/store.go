// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"time"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

type rescanType bool

// list of scanning modes
const (
	RescanVerified   rescanType = true
	NoRescanVerified rescanType = true
)

// StoreIncoming - store an incoming block checking to make sure it is valid first
func StoreIncoming(packedBlock, packedNextBlock []byte, performRescan rescanType) (err error) {
	start := time.Now()

	globalData.Lock()
	defer globalData.Unlock()

	reservoir.Disable()
	defer reservoir.Enable()

	reservoir.ClearSpend()

	// get current block header
	height, previousBlock, previousVersion, previousTimestamp := blockheader.Get()

	shouldFastSync := packedNextBlock != nil

	// extract incoming block record, checking for correct sequence
	var digest blockdigest.Digest
	var header *blockrecord.Header
	var data []byte

	br := blockrecord.Get()

	if shouldFastSync {
		h, _, d, err := br.ExtractHeader(packedBlock, height+1, true)
		if nil != err {
			globalData.log.Errorf("extract header error: %s", err)
			return err
		}
		nextH, _, _, err := br.ExtractHeader(packedNextBlock, height+2, true)
		if nil != err {
			globalData.log.Errorf("extract header error: %s", err)
			return err
		}
		header = h
		digest = nextH.PreviousBlock
		data = d
	} else {
		h, di, d, err := br.ExtractHeader(packedBlock, height+1, false)
		if nil != err {
			globalData.log.Errorf("extract header error: %s", err)
			return err
		}
		header = h
		digest = di
		data = d
	}

	// incoming version should always be larger or equal to current
	if err := blockrecord.ValidHeaderVersion(previousVersion, header.Version); err != nil {
		return err
	}

	if blockrecord.IsBlockToAdjustDifficulty(header.Number, header.Version) {
		nextDifficulty, prevDifficulty, err := blockrecord.AdjustDifficultyAtBlock(header.Number)
		// if any error happens for storing block, reset difficulty back to old value
		defer func(prevDifficulty float64) {
			if err != nil {
				difficulty.Current.Set(prevDifficulty)
			}
		}(prevDifficulty)

		if err != nil {
			globalData.log.Errorf("adjust difficulty with error: %s", err)
			return err
		}
		globalData.log.Infof("previous difficulty: %f, current difficulty: %f", prevDifficulty, nextDifficulty)
	}

	if err := blockrecord.ValidIncomingDifficulty(header, mode.ChainName()); err != nil {
		globalData.log.Errorf("incoming block difficulty %f different from local %f", header.Difficulty.Value(), difficulty.Current.Value())
		return err
	}

	if !shouldFastSync {
		if ok := digest.IsValidByDifficulty(header.Difficulty, mode.ChainName()); !ok {
			globalData.log.Warnf("digest error: %s", fault.InvalidBlockHeaderDifficulty)
			return fault.InvalidBlockHeaderDifficulty
		}

		// ensure correct linkage
		if err := blockrecord.ValidBlockLinkage(previousBlock, header.PreviousBlock); err != nil {
			return err
		}
	}

	// create database key for block number
	thisBlockNumberKey := make([]byte, 8)
	binary.BigEndian.PutUint64(thisBlockNumberKey, header.Number)

	// timestamp must be higher than previous
	if previousTimestamp > header.Timestamp {
		d := previousTimestamp - header.Timestamp
		globalData.log.Warnf("prev: %d  next: %d  diff: %d  block: %d  version: %d", previousTimestamp, header.Timestamp, d, header.Number, header.Version)

		if err := blockrecord.ValidBlockTimeSpacingAtVersion(header.Version, d); err != nil {
			return err
		}
	}

	// to overcome problem in V1 header blocks
	suppressDuplicateRecordChecks := header.Version == 1

	// extract the transaction data
	type txn struct {
		txId           merkle.Digest
		packed         transactionrecord.Packed
		unpacked       transactionrecord.Transaction
		linkOwner      *account.Account
		blockNumberKey []byte
	}

	txs := make([]txn, header.TransactionCount)

	// transaction validation (must return error and not panic)
	// ========================================================
	{
		// this is to double check the merkle root
		txIds := make([]merkle.Digest, header.TransactionCount)

		localAssets := make(map[transactionrecord.AssetIdentifier]struct{})

		// check all transactions are valid
		for i := uint16(0); i < header.TransactionCount; i++ {
			transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
			if nil != err {
				return err
			}
			txId := merkle.NewDigest(data[:n])

			// repack records to check signature is valid
			switch tx := transaction.(type) {

			case *transactionrecord.OldBaseData:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}

			case *transactionrecord.AssetData:
				_, err := tx.Pack(tx.Registrant)
				if nil != err {
					return err
				}
				assetId := tx.AssetId()
				if !suppressDuplicateRecordChecks && storage.Pool.Assets.Has(assetId[:]) {
					return fault.TransactionAlreadyExists
				}
				localAssets[assetId] = struct{}{}

			case *transactionrecord.BitmarkIssue:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}
				if !suppressDuplicateRecordChecks && storage.Pool.Transactions.Has(txId[:]) {
					return fault.TransactionAlreadyExists
				}
				if _, ok := localAssets[tx.AssetId]; !ok {
					if !storage.Pool.Assets.Has(tx.AssetId[:]) {
						return fault.AssetNotFound
					}
				}

			case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
				tr := tx.(transactionrecord.BitmarkTransfer)
				link := tr.GetLink()
				_, linkOwner := ownership.OwnerOf(nil, link)
				if nil == linkOwner {
					return fault.LinkToInvalidOrUnconfirmedTransaction
				}
				_, err := tx.Pack(linkOwner)
				if nil != err {
					return err
				}

				if !ownership.CurrentlyOwns(nil, linkOwner, link, storage.Pool.OwnerTxIndex) {
					return fault.DoubleTransferAttempt
				}

				ownerData, err := ownership.GetOwnerData(nil, link, storage.Pool.OwnerData)
				if nil != err {
					return fault.DoubleTransferAttempt
				}
				_, ok := ownerData.(*ownership.ShareOwnerData)
				if ok {
					return fault.CannotConvertSharesBackToAssets
				}

				txs[i].linkOwner = linkOwner

			case *transactionrecord.BlockFoundation:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}

			case *transactionrecord.BlockOwnerTransfer:
				link := tx.Link
				_, linkOwner := ownership.OwnerOf(nil, link)
				_, err = tx.Pack(linkOwner)
				if nil != err {
					return err
				}
				if !ownership.CurrentlyOwns(nil, linkOwner, link, storage.Pool.OwnerTxIndex) {
					return fault.DoubleTransferAttempt
				}

				// get the block number that is being transferred by this record
				thisBN := storage.Pool.BlockOwnerTxIndex.Get(link[:])
				if nil == thisBN {
					return fault.LinkToInvalidOrUnconfirmedTransaction
				}

				err = transactionrecord.CheckPayments(tx.Version, mode.IsTesting(), tx.Payments)
				if nil != err {
					return err
				}

				txs[i].blockNumberKey = thisBN
				txs[i].linkOwner = linkOwner

			case *transactionrecord.BitmarkShare:
				link := tx.Link
				_, linkOwner := ownership.OwnerOf(nil, link)
				if nil == linkOwner {
					return fault.LinkToInvalidOrUnconfirmedTransaction
				}
				_, err := tx.Pack(linkOwner)
				if nil != err {
					return err
				}

				ownerData, err := ownership.GetOwnerData(nil, link, storage.Pool.OwnerData)
				if nil != err {
					return fault.DoubleTransferAttempt
				}
				_, ok := ownerData.(*ownership.AssetOwnerData)
				if !ok {
					return fault.CanOnlyConvertAssetsToShares
				}

				txs[i].linkOwner = linkOwner

			case *transactionrecord.ShareGrant:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}
				_, err = reservoir.CheckGrantBalance(nil, tx, storage.Pool.ShareQuantity)
				if nil != err {
					return err
				}

			case *transactionrecord.ShareSwap:
				_, err := tx.Pack(tx.OwnerOne)
				if nil != err {
					return err
				}
				_, _, err = reservoir.CheckSwapBalances(nil, tx, storage.Pool.ShareQuantity)
				if nil != err {
					return err
				}

			default:
				// occurs if the above code is not in sync with transactionrecord/unpack.go
				// i.e. one or more case blocks are missing
				//      above _MUST_ code all transaction types
				// (this is the only panic condition in the validation code)
				globalData.log.Errorf("unhandled transaction: %v", tx)
				logger.Panicf("block/store: unhandled transaction: %v", tx)
			}

			txs[i].txId = txId
			txs[i].packed = transactionrecord.Packed(data[:n])
			txs[i].unpacked = transaction
			txIds[i] = txId
			data = data[n:]

			// fail if extraneous data after final transaction
			if i+1 == header.TransactionCount && len(data) > 0 {
				return fault.TransactionCountOutOfRange
			}
		}

		// build the tree of transaction IDs
		fullMerkleTree := merkle.FullMerkleTree(txIds)
		merkleRoot := fullMerkleTree[len(fullMerkleTree)-1]

		if merkleRoot != header.MerkleRoot {
			return fault.MerkleRootDoesNotMatch
		}

	} // end of validation

	// update database code, errors can cause panic
	// ============================================

	// create the ownership record
	var packedPayments []byte
	var packedFoundation []byte
	var blockOwner *account.Account
	txStart := 1
	// ensure the first transaction is base or owner
	switch tx := txs[0].unpacked.(type) {

	case *transactionrecord.BlockFoundation:
		err := transactionrecord.CheckPayments(tx.Version, mode.IsTesting(), tx.Payments)
		if nil != err {
			return err
		}
		packedPayments, err = tx.Payments.Pack(mode.IsTesting())
		if nil != err {
			return err
		}
		packedFoundation = txs[0].packed
		blockOwner = tx.Owner

	case *transactionrecord.OldBaseData:
		err := tx.Currency.ValidateAddress(tx.PaymentAddress, mode.IsTesting())
		if nil != err {
			return err
		}
		currencies := make(currency.Map)
		currencies[tx.Currency] = tx.PaymentAddress

		if tx1, ok := txs[1].unpacked.(*transactionrecord.OldBaseData); ok {
			// second tx is another base record
			currencies[tx1.Currency] = tx1.PaymentAddress
			txStart = 2
			packedFoundation = append(txs[0].packed, txs[1].packed...)
		} else {
			// else if single base block generate corresponding Litecoin address
			currencies[currency.Litecoin], _ = litecoin.FromBitcoin(tx.PaymentAddress)
			packedFoundation = txs[0].packed
		}
		packedPayments, err = currencies.Pack(mode.IsTesting())
		if nil != err {
			return err
		}
		blockOwner = tx.Owner

	default:
		return fault.MissingBlockOwner
	}

	if len(txs) < txStart {
		return fault.TransactionCountOutOfRange
	}

	trx, err := storage.NewDBTransaction()
	if nil != err {
		return err
	}

	// process the transactions into the database
	// but skip base/block-issue as these are already processed
	for _, item := range txs[txStart:] {
		//txId := item.txId
		//packed := item.packed

		switch tx := item.unpacked.(type) {

		case *transactionrecord.OldBaseData:
			// already processed
			trx.Abort()
			logger.Panicf("should not occur: %+v", tx)

		case *transactionrecord.AssetData:
			assetId := tx.AssetId()
			asset.Delete(assetId) // delete from pending cache
			assets := storage.Pool.Assets
			if !trx.Has(assets, assetId[:]) {
				trx.Put(assets, assetId[:], thisBlockNumberKey, item.packed)
			}

		case *transactionrecord.BitmarkIssue:
			reservoir.DeleteByTxId(item.txId) // delete from pending cache
			issues := storage.Pool.Transactions
			if !trx.Has(issues, item.txId[:]) {
				trx.Put(issues, item.txId[:], thisBlockNumberKey, item.packed)
				ownership.CreateAsset(
					trx,
					item.txId,
					header.Number,
					tx.AssetId,
					tx.Owner,
				)
			}

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			tr := tx.(transactionrecord.BitmarkTransfer)
			reservoir.DeleteByTxId(item.txId)
			link := tr.GetLink()

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(link)

			txrs := storage.Pool.Transactions
			trx.Put(txrs, item.txId[:], thisBlockNumberKey, item.packed)
			ownership.Transfer(trx,
				link,
				item.txId,
				header.Number,
				item.linkOwner,
				tr.GetOwner(),
			)

		case *transactionrecord.BlockFoundation:
			trx.Abort()
			// already processed
			logger.Panicf("should not occur: %+v", tx)

		case *transactionrecord.BlockOwnerTransfer:
			reservoir.DeleteByTxId(item.txId)
			link := tx.Link

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(link)

			// payments for the block being transferred
			// not to be confused with this block's packed payments
			pkPayments, err := tx.Payments.Pack(mode.IsTesting())
			if nil != err {
				trx.Abort()
				// packing was checked earlier, an error here is memory corruption
				logger.Panicf("pack, should not error: %s", err)
			}

			txrs := storage.Pool.Transactions
			trx.Put(txrs, item.txId[:], thisBlockNumberKey, item.packed)
			trx.Put(
				storage.Pool.BlockOwnerPayment,
				item.blockNumberKey,
				pkPayments,
				[]byte{},
			)
			trx.Put(
				storage.Pool.BlockOwnerTxIndex,
				item.txId[:],
				item.blockNumberKey,
				[]byte{},
			)
			trx.Delete(storage.Pool.BlockOwnerTxIndex, link[:])
			ownership.Transfer(trx, link, item.txId, header.Number, item.linkOwner, tx.Owner)

		case *transactionrecord.BitmarkShare:

			reservoir.DeleteByTxId(item.txId)
			link := tx.Link

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(link)

			txrs := storage.Pool.Transactions
			trx.Put(txrs, item.txId[:], thisBlockNumberKey, item.packed)
			ownership.Share(trx, link, item.txId, header.Number, item.linkOwner, tx.Quantity)

		case *transactionrecord.ShareGrant:

			reservoir.DeleteByTxId(item.txId)

			oKey := append(tx.Owner.Bytes(), tx.ShareId[:]...)
			rKey := append(tx.Recipient.Bytes(), tx.ShareId[:]...)

			oAccountBalance, ok := trx.GetN(storage.Pool.ShareQuantity, oKey)
			if !ok {
				trx.Abort()
				// check was earlier
				logger.Panic("read owner balance should not fail")
			}

			// if record does not exists the balance is zero
			rAccountBalance, _ := trx.GetN(storage.Pool.ShareQuantity, rKey)

			// owner, share → recipient
			oAccountBalance -= tx.Quantity
			rAccountBalance += tx.Quantity

			share := storage.Pool.ShareQuantity

			// update balances
			if 0 == oAccountBalance {
				trx.Delete(share, oKey)
			} else {
				trx.PutN(share, oKey, oAccountBalance)
			}
			trx.PutN(share, rKey, rAccountBalance)

			trx.Put(
				storage.Pool.Transactions,
				item.txId[:],
				thisBlockNumberKey,
				item.packed,
			)

		case *transactionrecord.ShareSwap:

			reservoir.DeleteByTxId(item.txId)

			ownerOneShareOneKey := append(tx.OwnerOne.Bytes(), tx.ShareIdOne[:]...)
			ownerOneShareTwoKey := append(tx.OwnerOne.Bytes(), tx.ShareIdTwo[:]...)
			ownerTwoShareOneKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdOne[:]...)
			ownerTwoShareTwoKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdTwo[:]...)

			ownerOneShareOneAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerOneShareOneKey)
			if !ok {
				trx.Abort()
				// check was earlier
				logger.Panic("read owner one share one balance should not fail")
			}

			ownerTwoShareTwoAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerTwoShareTwoKey)
			if !ok {
				trx.Abort()
				// check was earlier
				logger.Panic("read owner two share two balance should not fail")
			}

			// if record does not exist the balance is zero
			ownerOneShareTwoAccountBalance, _ := trx.GetN(
				storage.Pool.ShareQuantity,
				ownerOneShareTwoKey,
			)
			ownerTwoShareOneAccountBalance, _ := trx.GetN(
				storage.Pool.ShareQuantity,
				ownerTwoShareOneKey,
			)

			// owner 1, share 1 → owner 2
			ownerOneShareOneAccountBalance -= tx.QuantityOne
			ownerTwoShareOneAccountBalance += tx.QuantityOne

			// owner 2, share 2 → owner 1
			ownerTwoShareTwoAccountBalance -= tx.QuantityTwo
			ownerOneShareTwoAccountBalance += tx.QuantityTwo

			share := storage.Pool.ShareQuantity
			// update database share one
			if 0 == ownerOneShareOneAccountBalance {
				trx.Delete(share, ownerOneShareOneKey)
			} else {
				trx.PutN(share, ownerOneShareOneKey, ownerOneShareOneAccountBalance)
			}
			trx.PutN(share, ownerTwoShareOneKey, ownerTwoShareOneAccountBalance)

			// update database share two
			if 0 == ownerTwoShareTwoAccountBalance {
				trx.Delete(share, ownerTwoShareTwoKey)
			} else {
				trx.PutN(share, ownerTwoShareTwoKey, ownerTwoShareTwoAccountBalance)
			}
			trx.PutN(share, ownerOneShareTwoKey, ownerOneShareTwoAccountBalance)
			trx.Put(
				storage.Pool.Transactions,
				item.txId[:],
				thisBlockNumberKey,
				item.packed,
			)

		default:
			trx.Abort()
			globalData.log.Criticalf("unhandled transaction: %v", tx)
			logger.Panicf("unhandled transaction: %v", tx)
		}
	}

	// payment data
	trx.Put(
		storage.Pool.BlockOwnerPayment,
		thisBlockNumberKey,
		packedPayments,
		[]byte{},
	)

	// create the foundation record
	foundationTxId := blockrecord.FoundationTxId(header.Number, digest)
	trx.Put(
		storage.Pool.Transactions,
		foundationTxId[:],
		thisBlockNumberKey,
		packedFoundation,
	)

	// current owner: either foundation or block owner transfer: tx id → owned block
	trx.Put(
		storage.Pool.BlockOwnerTxIndex,
		foundationTxId[:],
		thisBlockNumberKey,
		[]byte{},
	)

	ownership.CreateBlock(trx, foundationTxId, header.Number, blockOwner)

	expectedBlockNumber := height + 1
	if expectedBlockNumber != header.Number {
		trx.Abort()
		logger.Panicf("block.Store: out of sequence block: actual: %d  expected: %d", header.Number, expectedBlockNumber)
	}

	blockheader.Set(header.Number, digest, header.Version, header.Timestamp)

	// return early if rebuilding, otherwise store and update DB
	if globalData.rebuild {
		globalData.log.Debugf("rebuilt block: %d time elapsed: %f", header.Number, time.Since(start).Seconds())
		trx.Commit()
		return nil
	}

	// finally store the block
	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	trx.Put(
		storage.Pool.Blocks,
		blockNumber,
		packedBlock,
		[]byte{},
	)

	trx.Put(
		storage.Pool.BlockHeaderHash,
		thisBlockNumberKey,
		digest[:],
		[]byte{},
	)

	globalData.log.Debugf("stored block: %d time elapsed: %f", header.Number, time.Since(start).Seconds())

	err = trx.Commit()
	if nil != err {
		return err
	}

	// rescan reservoir to drop any invalid transactions
	if performRescan {
		reservoir.Rescan()
	}

	return nil
}
