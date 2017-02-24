// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
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

	if globalData.previousBlock != header.PreviousBlock {
		return fault.ErrPreviousBlockDigestDoesNotMatch
	}

	data := packedBlock[blockrecord.TotalBlockSize:]

	type txn struct {
		packed   transactionrecord.Packed
		unpacked interface{}
	}

	txs := make([]txn, header.TransactionCount)
	txIds := make([]merkle.Digest, header.TransactionCount)

	// check all transactions are valid
	for i := uint16(0); i < header.TransactionCount; i += 1 {
		transaction, n, err := transactionrecord.Packed(data).Unpack()
		if nil != err {
			return err
		}

		txs[i].packed = transactionrecord.Packed(data[:n])
		txs[i].unpacked = transaction
		txIds[i] = merkle.NewDigest(data[:n])
		data = data[n:]
	}

	// build the tree of transaction IDs
	fullMerkleTree := merkle.FullMerkleTree(txIds)
	merkleRoot := fullMerkleTree[len(fullMerkleTree)-1]

	if merkleRoot != header.MerkleRoot {
		return fault.ErrMerkleRootDoesNotMatch
	}

	digest := packedHeader.Digest()
	storeAndUpdate(header, digest, packedBlock)

	// store transactions
	for i, item := range txs {
		txId := txIds[i]
		packed := item.packed
		switch tx := item.unpacked.(type) {

		case *transactionrecord.BaseData:
			blockNumber := make([]byte, 8)
			binary.BigEndian.PutUint64(blockNumber, header.Number)
			data := make([]byte, 8, 8+len(tx.PaymentAddress))
			binary.BigEndian.PutUint64(data[:8], tx.Currency.Uint64())
			data = append(data, tx.PaymentAddress...)
			storage.Pool.BlockOwners.Put(blockNumber, data)
			// currently not stored separately

		case *transactionrecord.AssetData:
			assetIndex := tx.AssetIndex()
			key := assetIndex[:]
			asset.Delete(assetIndex)
			storage.Pool.Assets.Put(key, packed)

		case *transactionrecord.BitmarkIssue:
			key := txId[:]
			reservoir.DeleteByTxId(txId)
			storage.Pool.Transactions.Put(key, packed)
			CreateOwnership(txId, header.Number, tx.AssetIndex, tx.Owner)

		case *transactionrecord.BitmarkTransfer:
			key := txId[:]
			reservoir.DeleteByTxId(txId)

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(tx.Link)

			storage.Pool.Transactions.Put(key, packed)
			linkOwner := OwnerOf(tx.Link)
			if nil == linkOwner {
				fault.Criticalf("missing transaction record for link: %v refererenced by tx id: %v", tx.Link, txId)
				fault.Panic("Transactions database is corrupt")
			}
			TransferOwnership(tx.Link, txId, header.Number, linkOwner, tx.Owner)

		default:
			globalData.log.Criticalf("unhandled transaction: %v", tx)
			fault.Panicf("unhandled transaction: %v", tx)
		}
	}

	return nil
}

// store the block and update block data
// hold lock before calling this
func storeAndUpdate(header *blockrecord.Header, digest blockdigest.Digest, packedBlock []byte) {

	expectedBlockNumber := globalData.height + 1
	if expectedBlockNumber != header.Number {
		fault.Panicf("block.Store: out of sequence block: actual: %d  expected: %d", header.Number, expectedBlockNumber)
	}

	globalData.previousBlock = digest
	globalData.height = header.Number

	blockring.Put(header.Number, digest, packedBlock)

	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	storage.Pool.Blocks.Put(blockNumber, packedBlock)
}
