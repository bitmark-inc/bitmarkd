// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// delete from current highest block down to and including the specified block
func DeleteDownToBlock(finalBlockNumber uint64) error {
	globalData.Lock()
	defer globalData.Unlock()

	log := globalData.log

	log.Infof("Delete down to block: %d", finalBlockNumber)

	last, ok := storage.Pool.Blocks.LastElement()
	if !ok {
		return nil // block store is already empty
	}

	packedBlock := last.Value

	for {
		packedHeader := blockrecord.PackedHeader(packedBlock[:blockrecord.TotalBlockSize])
		header, err := packedHeader.Unpack()
		if nil != err {
			log.Criticalf("failed to unpack block: %d from storage  error: %v", binary.BigEndian.Uint64(last.Key), err)
			return err
		}

		// finished
		if header.Number < finalBlockNumber {
			log.Infof("finish: _NOT_ Deleting: %d", header.Number)
			clearRingBuffer(log)
			return nil
		}

		log.Infof("Delete block: %d  transactions: %d", header.Number, header.TransactionCount)

		// packed transactions
		data := packedBlock[blockrecord.TotalBlockSize:]
	loop:
		for i := 1; true; i += 1 {
			transaction, n, err := transactionrecord.Packed(data).Unpack()
			if nil != err {
				log.Errorf("tx[%d]: error: %v", i, err)
				return err
			}

			packedTransaction := transactionrecord.Packed(data[:n])
			switch transaction.(type) {
			case *transactionrecord.BaseData:
				// currently not stored separately

			case *transactionrecord.AssetData:
				asset := transaction.(*transactionrecord.AssetData)
				assetIndex := asset.AssetIndex()
				key := assetIndex[:]
				storage.Pool.Assets.Delete(key)
				storage.Pool.VerifiedAssets.Put(key, packedTransaction)

			case *transactionrecord.BitmarkIssue:
				issue := transaction.(*transactionrecord.BitmarkIssue)
				txId := packedTransaction.MakeLink()
				key := txId[:]
				storage.Pool.Transactions.Delete(key)
				storage.Pool.VerifiedTransactions.Put(key, packedTransaction)
				TransferOwnership(txId, txId, 0, issue.Owner, nil)

			case *transactionrecord.BitmarkTransfer:
				transfer := transaction.(*transactionrecord.BitmarkTransfer)
				txId := packedTransaction.MakeLink()
				key := txId[:]
				storage.Pool.Transactions.Delete(key)
				storage.Pool.VerifiedTransactions.Put(key, packedTransaction)
				linkOwner := OwnerOf(transfer.Link)
				if nil == linkOwner {
					log.Criticalf("missing transaction record for: %v", transfer.Link)
					fault.Panic("Transactions database is corrupt")
				}
				// just use zero here, as the fork restore should overwrite with new chain, incluing updated block number
				// ***** FIX THIS: is the above statement sufficient
				TransferOwnership(txId, transfer.Link, 0, transfer.Owner, linkOwner)

			default:
				fault.Panicf("unexpected transaction: %v", transaction)
			}

			data = data[n:]
			if 0 == len(data) {
				break loop
			}
		}

		// delete the block data
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, header.Number)
		storage.Pool.Blocks.Delete(key)

		// fetch previous block number
		binary.BigEndian.PutUint64(key, header.Number-1)
		packedBlock = storage.Pool.Blocks.Get(key)

		if nil == packedBlock {
			break
		}

	}
	return nil
}
