// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
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

	reservoir.Disable()
	defer reservoir.Enable()

	packedBlock := last.Value

outer_loop:
	for {
		header, digest, data, err := blockrecord.ExtractHeader(packedBlock)
		if nil != err {
			log.Criticalf("failed to unpack block: %d from storage  error: %s", binary.BigEndian.Uint64(last.Key), err)
			return err
		}

		// finished
		if header.Number < finalBlockNumber {
			log.Infof("finish: _NOT_ Deleting: %d", header.Number)
			fillRingBuffer(log)
			return nil
		}

		log.Infof("Delete block: %d  transactions: %d", header.Number, header.TransactionCount)

		// record block owner
		var blockOwner *account.Account

		// handle packed transactions
	inner_loop:
		for i := 1; true; i += 1 {
			transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
			if nil != err {
				log.Errorf("tx[%d]: error: %s", i, err)
				return err
			}

			packedTransaction := transactionrecord.Packed(data[:n])
			switch tx := transaction.(type) {
			case *transactionrecord.OldBaseData:
				if nil == blockOwner {
					blockOwner = tx.Owner
				}
				// delete later

			case *transactionrecord.AssetData:
				assetId := tx.AssetId()
				storage.Pool.Assets.Delete(assetId[:])
				asset.Delete(assetId)

			case *transactionrecord.BitmarkIssue:
				txId := packedTransaction.MakeLink()
				reservoir.DeleteByTxId(txId)
				if storage.Pool.Transactions.Has(txId[:]) {
					storage.Pool.Transactions.Delete(txId[:])
					ownership.Transfer(txId, txId, 0, tx.Owner, nil)
				}

			case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
				tr := tx.(transactionrecord.BitmarkTransfer)
				txId := packedTransaction.MakeLink()
				storage.Pool.Transactions.Delete(txId[:])
				reservoir.DeleteByTxId(txId)
				link := tr.GetLink()
				linkOwner := ownership.OwnerOf(link)
				if nil == linkOwner {
					log.Criticalf("missing transaction record for: %v", link)
					logger.Panic("Transactions database is corrupt")
				}
				// just use zero here, as the fork restore should overwrite with new chain, including updated block number
				ownership.Transfer(txId, link, 0, tr.GetOwner(), linkOwner)

			case *transactionrecord.BlockFoundation:
				if nil == blockOwner {
					blockOwner = tx.Owner
				}
				// delete later

			case *transactionrecord.BlockOwnerTransfer:
				txId := packedTransaction.MakeLink()
				key := txId[:]
				storage.Pool.Transactions.Delete(key)
				reservoir.DeleteByTxId(txId)
				linkOwner := ownership.OwnerOf(tx.Link)
				if nil == linkOwner {
					log.Criticalf("missing transaction record for: %v", tx.Link)
					logger.Panic("Transactions database is corrupt")
				}
				// just use zero here, as the fork restore should overwrite with new chain, including updated block number
				ownership.Transfer(txId, tx.Link, 0, tx.Owner, linkOwner)

			default:
				logger.Panicf("unexpected transaction: %v", transaction)
			}

			data = data[n:]
			if 0 == len(data) {
				break inner_loop
			}
		}

		// block number key for deletion
		blockNumberKey := make([]byte, 8)
		binary.BigEndian.PutUint64(blockNumberKey, header.Number)

		// block ownership remove
		foundationTxId := blockrecord.FoundationTxId(header, digest)
		storage.Pool.Transactions.Delete(foundationTxId[:])
		if nil == blockOwner {
			log.Criticalf("nil block owner for block: %d", header.Number)
		} else {
			ownership.Transfer(foundationTxId, foundationTxId, 0, blockOwner, nil)
		}
		// remove remaining block data
		storage.Pool.BlockOwnerTxIndex.Delete(foundationTxId[:])
		storage.Pool.Blocks.Delete(blockNumberKey)

		// fetch previous block number
		binary.BigEndian.PutUint64(blockNumberKey, header.Number-1)
		packedBlock = storage.Pool.Blocks.Get(blockNumberKey)

		if nil == packedBlock {
			break outer_loop
		}

	}
	return nil
}
