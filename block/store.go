// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// store the block and update block data
func store(header *blockrecord.Header, digest blockdigest.Digest, packedBlock []byte) {
	globalData.Lock()
	//defer globalData.Unlock()

	expectedBlockNumber := globalData.height + 1
	if expectedBlockNumber != header.Number {
		fault.Panicf("block.Store: out of sequence block: actual: %d  expected: %d", header.Number, expectedBlockNumber)
	}

	globalData.previousBlock = digest
	globalData.height = header.Number

	i := globalData.ringIndex
	globalData.ring[i].number = header.Number
	globalData.ring[i].digest = digest
	globalData.ring[i].crc = CRC(header.Number, packedBlock)
	i = i + 1
	if i >= len(globalData.ring) {
		i = 0
	}
	globalData.ringIndex = i

	// end of critical section
	globalData.Unlock()

	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	storage.Pool.Blocks.Put(blockNumber, packedBlock)
}

// store an incoming block checking to make sure it is valid first
func StoreIncoming(packedBlock []byte) error {
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
	store(header, digest, packedBlock)

	// store transactions
	for i, txn := range txs {
		txid := txIds[i]
		packed := txn.packed
		transaction := txn.unpacked
		switch transaction.(type) {

		case *transactionrecord.BaseData:
			base := transaction.(*transactionrecord.BaseData)
			blockNumber := make([]byte, 8)
			binary.BigEndian.PutUint64(blockNumber, header.Number)
			data := make([]byte, 8, 8+len(base.PaymentAddress))
			binary.BigEndian.PutUint64(data[:8], base.Currency.Uint64())
			data = append(data, base.PaymentAddress...)
			storage.Pool.BlockOwners.Put(blockNumber, data)
			// currently not stored separately

		case *transactionrecord.AssetData:
			asset := transaction.(*transactionrecord.AssetData)
			assetIndex := asset.AssetIndex()
			key := assetIndex[:]
			storage.Pool.VerifiedAssets.Delete(key)
			storage.Pool.Assets.Put(key, packed)

		case *transactionrecord.BitmarkIssue:
			issue := transaction.(*transactionrecord.BitmarkIssue)
			key := txid[:]
			storage.Pool.VerifiedTransactions.Delete(key)
			storage.Pool.Transactions.Put(key, packed)
			CreateOwnership(txid, issue.AssetIndex, header.Number, issue.Owner)

		case *transactionrecord.BitmarkTransfer:
			transfer := transaction.(*transactionrecord.BitmarkTransfer)
			key := txid[:]
			storage.Pool.VerifiedTransactions.Delete(key)
			storage.Pool.Transactions.Put(key, packed)
			linkOwner := OwnerOf(transfer.Link)
			if nil == linkOwner {
				fault.Criticalf("missing transaction record for: %v", transfer.Link)
				fault.Panic("Transactions database is corrupt")
			}
			TransferOwnership(txid, linkOwner, transfer.Owner)

		default:
			globalData.log.Criticalf("unhandled transaction: %v", transaction)
			fault.Panicf("unhandled transaction: %v", transaction)
		}
	}

	return nil
}
