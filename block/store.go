// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"time"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
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
func StoreIncoming(packedBlock []byte, performRescan rescanType) error {
	start := time.Now()

	globalData.Lock()
	defer globalData.Unlock()

	reservoir.Disable()
	defer reservoir.Enable()

	reservoir.ClearSpend()

	// get current block header
	height, previousBlock, previousVersion, previousTimestamp := blockheader.Get()

	// extract incoming block record, checking for correct sequence
	header, digest, data, err := blockrecord.ExtractHeader(packedBlock, height+1)
	if nil != err {
		return err
	}

	// ensure correct linkage
	if previousBlock != header.PreviousBlock {
		return fault.ErrPreviousBlockDigestDoesNotMatch
	}

	// check version
	if header.Version < 1 {
		return fault.ErrInvalidBlockHeaderVersion
	}

	// block version must be the same or higher
	if previousVersion > header.Version {
		return fault.ErrBlockVersionMustNotDecrease
	}

	// create database key for block number
	thisBlockNumberKey := make([]byte, 8)
	binary.BigEndian.PutUint64(thisBlockNumberKey, header.Number)

	// timestamp must be higher than previous
	if previousTimestamp > header.Timestamp {
		d := previousTimestamp - header.Timestamp
		globalData.log.Warnf("prev: %d  next: %d  diff: %d  block: %d  version: %d", previousTimestamp, header.Timestamp, d, header.Number, header.Version)

		// allow more tolerance for old blocks up to a few minutes back in time
		fail := false
		switch header.Version {
		case 1:
			fail = d > 240*60 // seconds
		case 2:
			fail = d > 10*60 // seconds
		default:
			fail = true
		}
		if fail {
			return fault.ErrInvalidBlockHeaderTimestamp
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
		for i := uint16(0); i < header.TransactionCount; i += 1 {
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
					return fault.ErrTransactionAlreadyExists
				}
				localAssets[assetId] = struct{}{}

			case *transactionrecord.BitmarkIssue:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}
				if !suppressDuplicateRecordChecks && storage.Pool.Transactions.Has(txId[:]) {
					return fault.ErrTransactionAlreadyExists
				}
				if _, ok := localAssets[tx.AssetId]; !ok {
					if !storage.Pool.Assets.Has(tx.AssetId[:]) {
						return fault.ErrAssetNotFound
					}
				}

			case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
				tr := tx.(transactionrecord.BitmarkTransfer)
				link := tr.GetLink()
				_, linkOwner := ownership.OwnerOf(link)
				if nil == linkOwner {
					return fault.ErrLinkToInvalidOrUnconfirmedTransaction
				}
				_, err := tx.Pack(linkOwner)
				if nil != err {
					return err
				}

				if !ownership.CurrentlyOwns(linkOwner, link) {
					return fault.ErrDoubleTransferAttempt
				}

				ownerData, err := ownership.GetOwnerData(link)
				if nil != err {
					return fault.ErrDoubleTransferAttempt
				}
				_, ok := ownerData.(*ownership.ShareOwnerData)
				if ok {
					return fault.ErrCannotConvertSharesBackToAssets
				}

				txs[i].linkOwner = linkOwner

			case *transactionrecord.BlockFoundation:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}

			case *transactionrecord.BlockOwnerTransfer:
				link := tx.Link
				_, linkOwner := ownership.OwnerOf(link)
				_, err = tx.Pack(linkOwner)
				if nil != err {
					return err
				}
				if !ownership.CurrentlyOwns(linkOwner, link) {
					return fault.ErrDoubleTransferAttempt
				}

				// get the block number that is being transferred by this record
				thisBN := storage.Pool.BlockOwnerTxIndex.Get(link[:])
				if nil == thisBN {
					return fault.ErrLinkToInvalidOrUnconfirmedTransaction
				}

				err = transactionrecord.CheckPayments(tx.Version, mode.IsTesting(), tx.Payments)
				if nil != err {
					return err
				}

				txs[i].blockNumberKey = thisBN
				txs[i].linkOwner = linkOwner

			case *transactionrecord.BitmarkShare:
				link := tx.Link
				_, linkOwner := ownership.OwnerOf(link)
				if nil == linkOwner {
					return fault.ErrLinkToInvalidOrUnconfirmedTransaction
				}
				_, err := tx.Pack(linkOwner)
				if nil != err {
					return err
				}

				ownerData, err := ownership.GetOwnerData(link)
				if nil != err {
					return fault.ErrDoubleTransferAttempt
				}
				_, ok := ownerData.(*ownership.AssetOwnerData)
				if !ok {
					return fault.ErrCanOnlyConvertAssetsToShares
				}

				txs[i].linkOwner = linkOwner

			case *transactionrecord.ShareGrant:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}
				_, err = reservoir.CheckGrantBalance(tx)
				if nil != err {
					return err
				}

			case *transactionrecord.ShareSwap:
				_, err := tx.Pack(tx.OwnerOne)
				if nil != err {
					return err
				}
				_, _, err = reservoir.CheckSwapBalances(tx)
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
				return fault.ErrTransactionCountOutOfRange
			}
		}

		// build the tree of transaction IDs
		fullMerkleTree := merkle.FullMerkleTree(txIds)
		merkleRoot := fullMerkleTree[len(fullMerkleTree)-1]

		if merkleRoot != header.MerkleRoot {
			return fault.ErrMerkleRootDoesNotMatch
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
			currencies[currency.Litecoin], err = litecoin.FromBitcoin(tx.PaymentAddress)
			packedFoundation = txs[0].packed
		}
		packedPayments, err = currencies.Pack(mode.IsTesting())
		if nil != err {
			return err
		}
		blockOwner = tx.Owner

	default:
		return fault.ErrMissingBlockOwner
	}

	if len(txs) < txStart {
		return fault.ErrTransactionCountOutOfRange
	}

	// process the transactions into the database
	// but skip base/block-issue as these are already processed
	for _, item := range txs[txStart:] {
		//txId := item.txId
		//packed := item.packed

		switch tx := item.unpacked.(type) {

		case *transactionrecord.OldBaseData:
			// already processed
			logger.Panicf("should not occur: %+v", tx)

		case *transactionrecord.AssetData:
			assetId := tx.AssetId()
			asset.Delete(assetId) // delete from pending cache
			if !storage.Pool.Assets.Has(assetId[:]) {
				storage.Pool.Assets.Put(assetId[:], thisBlockNumberKey, item.packed)
			}

		case *transactionrecord.BitmarkIssue:
			reservoir.DeleteByTxId(item.txId) // delete from pending cache
			if !storage.Pool.Transactions.Has(item.txId[:]) {
				storage.Pool.Transactions.Put(item.txId[:], thisBlockNumberKey, item.packed)
				ownership.CreateAsset(item.txId, header.Number, tx.AssetId, tx.Owner)
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

			storage.Pool.Transactions.Put(item.txId[:], thisBlockNumberKey, item.packed)
			ownership.Transfer(link, item.txId, header.Number, item.linkOwner, tr.GetOwner())

		case *transactionrecord.BlockFoundation:
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
				// packing was checked earlier, an error here is memory corruption
				logger.Panicf("pack, should not error: %s", err)
			}

			storage.Pool.Transactions.Put(item.txId[:], thisBlockNumberKey, item.packed)
			storage.Pool.BlockOwnerPayment.Put(item.blockNumberKey, pkPayments)
			storage.Pool.BlockOwnerTxIndex.Put(item.txId[:], item.blockNumberKey)
			storage.Pool.BlockOwnerTxIndex.Delete(link[:])
			ownership.Transfer(link, item.txId, header.Number, item.linkOwner, tx.Owner)

		case *transactionrecord.BitmarkShare:

			reservoir.DeleteByTxId(item.txId)
			link := tx.Link

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(link)

			storage.Pool.Transactions.Put(item.txId[:], thisBlockNumberKey, item.packed)
			ownership.Share(link, item.txId, header.Number, item.linkOwner, tx.Quantity)

		case *transactionrecord.ShareGrant:

			reservoir.DeleteByTxId(item.txId)

			oKey := append(tx.Owner.Bytes(), tx.ShareId[:]...)
			rKey := append(tx.Recipient.Bytes(), tx.ShareId[:]...)

			oAccountBalance, ok := storage.Pool.ShareQuantity.GetN(oKey)
			if !ok {
				// check was earlier
				logger.Panic("read owner balance should not fail")
			}

			// if record does not exists the balance is zero
			rAccountBalance, _ := storage.Pool.ShareQuantity.GetN(rKey)

			// owner, share → recipient
			oAccountBalance -= tx.Quantity
			rAccountBalance += tx.Quantity

			// update balances
			if 0 == oAccountBalance {
				storage.Pool.ShareQuantity.Delete(oKey)
			} else {
				storage.Pool.ShareQuantity.PutN(oKey, oAccountBalance)
			}
			storage.Pool.ShareQuantity.PutN(rKey, rAccountBalance)

			storage.Pool.Transactions.Put(item.txId[:], thisBlockNumberKey, item.packed)

		case *transactionrecord.ShareSwap:

			reservoir.DeleteByTxId(item.txId)

			ownerOneShareOneKey := append(tx.OwnerOne.Bytes(), tx.ShareIdOne[:]...)
			ownerOneShareTwoKey := append(tx.OwnerOne.Bytes(), tx.ShareIdTwo[:]...)
			ownerTwoShareOneKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdOne[:]...)
			ownerTwoShareTwoKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdTwo[:]...)

			ownerOneShareOneAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerOneShareOneKey)
			if !ok {
				// check was earlier
				logger.Panic("read owner one share one balance should not fail")
			}

			ownerTwoShareTwoAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerTwoShareTwoKey)
			if !ok {
				// check was earlier
				logger.Panic("read owner two share two balance should not fail")
			}

			// if record does not exist the balance is zero
			ownerOneShareTwoAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerOneShareTwoKey)
			ownerTwoShareOneAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerTwoShareOneKey)

			// owner 1, share 1 → owner 2
			ownerOneShareOneAccountBalance -= tx.QuantityOne
			ownerTwoShareOneAccountBalance += tx.QuantityOne

			// owner 2, share 2 → owner 1
			ownerTwoShareTwoAccountBalance -= tx.QuantityTwo
			ownerOneShareTwoAccountBalance += tx.QuantityTwo

			// update database share one
			if 0 == ownerOneShareOneAccountBalance {
				storage.Pool.ShareQuantity.Delete(ownerOneShareOneKey)
			} else {
				storage.Pool.ShareQuantity.PutN(ownerOneShareOneKey, ownerOneShareOneAccountBalance)
			}
			storage.Pool.ShareQuantity.PutN(ownerTwoShareOneKey, ownerTwoShareOneAccountBalance)

			// update database share two
			if 0 == ownerTwoShareTwoAccountBalance {
				storage.Pool.ShareQuantity.Delete(ownerTwoShareTwoKey)
			} else {
				storage.Pool.ShareQuantity.PutN(ownerTwoShareTwoKey, ownerTwoShareTwoAccountBalance)
			}
			storage.Pool.ShareQuantity.PutN(ownerOneShareTwoKey, ownerOneShareTwoAccountBalance)

			storage.Pool.Transactions.Put(item.txId[:], thisBlockNumberKey, item.packed)

		default:
			globalData.log.Criticalf("unhandled transaction: %v", tx)
			logger.Panicf("unhandled transaction: %v", tx)
		}
	}

	// payment data
	storage.Pool.BlockOwnerPayment.Put(thisBlockNumberKey, packedPayments)

	// create the foundation record
	foundationTxId := blockrecord.FoundationTxId(header, digest)
	storage.Pool.Transactions.Put(foundationTxId[:], thisBlockNumberKey, packedFoundation)

	// current owner: either foundation or block owner transfer: tx id → owned block
	storage.Pool.BlockOwnerTxIndex.Put(foundationTxId[:], thisBlockNumberKey)

	ownership.CreateBlock(foundationTxId, header.Number, blockOwner)

	expectedBlockNumber := height + 1
	if expectedBlockNumber != header.Number {
		logger.Panicf("block.Store: out of sequence block: actual: %d  expected: %d", header.Number, expectedBlockNumber)
	}

	blockheader.Set(header.Number, digest, header.Version, header.Timestamp)

	// return early if rebuilding, otherwise store and update DB
	if globalData.rebuild {
		globalData.log.Debugf("rebuilt block: %d time elapsed: %f", header.Number, time.Since(start).Seconds())
		return nil
	}

	// finally store the block
	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	storage.Pool.Blocks.Put(blockNumber, packedBlock)
	storage.Pool.BlockHeaderHash.Put(thisBlockNumberKey, digest[:])
	globalData.log.Debugf("stored block: %d time elapsed: %f", header.Number, time.Since(start).Seconds())

	// rescan reservoir to drop any invalid transactions
	if performRescan {
		reservoir.Rescan()
	}

	return nil
}
