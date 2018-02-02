// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// store an incoming block checking to make sure it is valid first
func StoreIncoming(packedBlock []byte) error {

	globalData.Lock()
	defer globalData.Unlock()

	reservoir.Disable()
	defer reservoir.Enable()

	packedHeader := blockrecord.PackedHeader(packedBlock[:blockrecord.TotalBlockSize])
	header, err := packedHeader.Unpack()
	if nil != err {
		return err
	}
	blockNumberKey := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumberKey, header.Number)

	if globalData.previousBlock != header.PreviousBlock {
		return fault.ErrPreviousBlockDigestDoesNotMatch
	}

	data := packedBlock[blockrecord.TotalBlockSize:]

	type txn struct {
		txId                   merkle.Digest
		packed                 transactionrecord.Packed
		unpacked               transactionrecord.Transaction
		linkOwner              *account.Account
		previousBlockNumberKey []byte
	}

	txs := make([]txn, header.TransactionCount)

	// transaction validator
	{
		// this is to double check the merkle root
		txIds := make([]merkle.Digest, header.TransactionCount)

		// check all transactions are valid
		for i := uint16(0); i < header.TransactionCount; i += 1 {
			transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
			if nil != err {
				return err
			}

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

			case *transactionrecord.BitmarkIssue:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}
			case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
				tr := tx.(transactionrecord.BitmarkTransfer)
				link := tr.GetLink()
				linkOwner := OwnerOf(link)
				if nil == linkOwner {
					logger.Criticalf("missing transaction record for link: %v refererenced by tx: %+v", link, tx)
					logger.Panic("Transactions database is corrupt")
				}
				_, err := tx.Pack(linkOwner)
				if nil != err {
					return err
				}
				txs[i].linkOwner = linkOwner

			case *transactionrecord.BlockFoundation:
				_, err := tx.Pack(tx.Owner)
				if nil != err {
					return err
				}

			case *transactionrecord.BlockOwnerTransfer:
				link := tx.Link
				linkOwner := OwnerOf(link)
				_, err = tx.Pack(linkOwner)
				if nil != err {
					return err
				}

				// get the block number that is being transferred by this record
				n := storage.Pool.BlockOwnerTxIndex.Get(link[:])
				if nil == n {
					globalData.log.Criticalf("missing BlockOwnerTxIndex: %v", link)
					logger.Panicf("missing BlockOwnerTxIndex: %v", link)
				}

				err = transactionrecord.CheckPayments(tx.Version, mode.IsTesting(), tx.Payments)
				if nil != err {
					return err
				}

				txs[i].previousBlockNumberKey = n
				txs[i].linkOwner = linkOwner

			default:
				globalData.log.Criticalf("unhandled transaction: %v", tx)
				logger.Panicf("unhandled transaction: %v", tx)
			}

			txId := merkle.NewDigest(data[:n])
			txs[i].txId = txId
			txs[i].packed = transactionrecord.Packed(data[:n])
			txs[i].unpacked = transaction
			txIds[i] = txId
			data = data[n:]
		}

		// build the tree of transaction IDs
		fullMerkleTree := merkle.FullMerkleTree(txIds)
		merkleRoot := fullMerkleTree[len(fullMerkleTree)-1]

		if merkleRoot != header.MerkleRoot {
			return fault.ErrMerkleRootDoesNotMatch
		}
	}

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

	// process the transactions into the database
	// but skip base/block-issue as these are already processed
	for _, item := range txs[txStart:] {
		//txId := item.txId
		//packed := item.packed

		switch tx := item.unpacked.(type) {

		case *transactionrecord.OldBaseData:
			logger.Panicf("should not occur: %+v", tx)

		case *transactionrecord.AssetData:
			assetIndex := tx.AssetIndex()
			asset.Delete(assetIndex)
			storage.Pool.Assets.Put(assetIndex[:], item.packed)

		case *transactionrecord.BitmarkIssue:
			reservoir.DeleteByTxId(item.txId)
			storage.Pool.Transactions.Put(item.txId[:], item.packed)
			CreateOwnership(item.txId, header.Number, tx.AssetIndex, tx.Owner)

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			tr := tx.(transactionrecord.BitmarkTransfer)
			reservoir.DeleteByTxId(item.txId)
			link := tr.GetLink()

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(link)

			storage.Pool.Transactions.Put(item.txId[:], item.packed)
			TransferOwnership(link, item.txId, header.Number, item.linkOwner, tr.GetOwner())

		case *transactionrecord.BlockFoundation:
			logger.Panicf("should not occur: %+v", tx)

		case *transactionrecord.BlockOwnerTransfer:
			reservoir.DeleteByTxId(item.txId)
			link := tx.Link

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(link)

			p, err := tx.Payments.Pack(mode.IsTesting())
			if nil != err {
				// packing was checked earlier, an error here is memory corruption
				logger.Panicf("pack, should not error: %s", err)
			}
			storage.Pool.BlockOwnerPayment.Put(item.previousBlockNumberKey, p)
			storage.Pool.BlockOwnerTxIndex.Put(item.txId[:], blockNumberKey)

		default:
			globalData.log.Criticalf("unhandled transaction: %v", tx)
			logger.Panicf("unhandled transaction: %v", tx)
		}
	}

	// payment data
	storage.Pool.BlockOwnerPayment.Put(blockNumberKey, packedPayments)

	// block digest
	digest := packedHeader.Digest()

	// create the foundation record
	foundationTxId := blockrecord.FoundationTxId(header, digest)
	storage.Pool.Transactions.Put(foundationTxId[:], packedFoundation)
	storage.Pool.BlockOwnerTxIndex.Put(foundationTxId[:], blockNumberKey)

	CreateOwnership(foundationTxId, header.Number, transactionrecord.AssetIndex{}, blockOwner)

	// finish be storing the block header
	storeAndUpdate(header, digest, packedBlock)

	return nil
}

// store the block and update block data
// hold lock before calling this
func storeAndUpdate(header *blockrecord.Header, digest blockdigest.Digest, packedBlock []byte) {

	expectedBlockNumber := globalData.height + 1
	if expectedBlockNumber != header.Number {
		logger.Panicf("block.Store: out of sequence block: actual: %d  expected: %d", header.Number, expectedBlockNumber)
	}

	globalData.previousBlock = digest
	globalData.height = header.Number

	blockring.Put(header.Number, digest, packedBlock)

	// finally store the block
	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	storage.Pool.Blocks.Put(blockNumber, packedBlock)
}
