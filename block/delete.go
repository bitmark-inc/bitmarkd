// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// DeleteDownToBlock - delete from current highest block down to and including the specified block
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
		header, digest, data, err := blockrecord.ExtractHeader(packedBlock, 0)
		if nil != err {
			log.Criticalf("failed to unpack block: %d from storage  error: %s", binary.BigEndian.Uint64(last.Key), err)
			return err
		}

		// finished
		if header.Number < finalBlockNumber {

			blockheader.Set(header.Number, digest, header.Version, header.Timestamp)

			log.Infof("finish: _NOT_ Deleting: %d", header.Number)
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
				blockNumber, linkOwner := ownership.OwnerOf(link)
				if nil == linkOwner {
					log.Criticalf("missing transaction record for: %v", link)
					logger.Panic("Transactions database is corrupt")
				}
				ownership.Transfer(txId, link, blockNumber, tr.GetOwner(), linkOwner)

			case *transactionrecord.BlockFoundation:
				if nil == blockOwner {
					blockOwner = tx.Owner
				}
				// delete later

			case *transactionrecord.BlockOwnerTransfer:
				txId := packedTransaction.MakeLink()
				storage.Pool.Transactions.Delete(txId[:])
				reservoir.DeleteByTxId(txId)
				blockNumber, linkOwner := ownership.OwnerOf(tx.Link)
				if nil == linkOwner {
					log.Criticalf("missing transaction record for: %v", tx.Link)
					logger.Panic("Transactions database is corrupt")
				}
				ownerdata, err := ownership.GetOwnerDataB(txId[:])
				if nil != err {
					log.Criticalf("missing ownership for: %s", txId)
					logger.Panic("Ownership database is corrupt")
				}
				blockOwnerdata, ok := ownerdata.(*ownership.BlockOwnerData)
				if !ok {
					log.Criticalf("expected block ownership but read: %+v", ownerdata)
					logger.Panic("Ownership database is corrupt")
				}

				ownership.Transfer(txId, tx.Link, blockNumber, tx.Owner, linkOwner)

				blockNumberKey := make([]byte, 8)
				binary.BigEndian.PutUint64(blockNumberKey, blockOwnerdata.IssueBlockNumber())

				// put block ownership back
				_, previous := storage.Pool.Transactions.GetNB(tx.Link[:])

				blockTransaction, _, err := transactionrecord.Packed(previous).Unpack(mode.IsTesting())
				if nil != err {
					logger.Criticalf("invalid error: %s", txId, err)
					logger.Panic("Transaction database is corrupt")
				}
				switch prevTx := blockTransaction.(type) {
				case *transactionrecord.BlockFoundation:
					err := transactionrecord.CheckPayments(prevTx.Version, mode.IsTesting(), prevTx.Payments)
					if nil != err {
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					packedPayments, err := prevTx.Payments.Pack(mode.IsTesting())
					if nil != err {
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					// payment data
					storage.Pool.BlockOwnerPayment.Put(blockNumberKey, packedPayments)
					storage.Pool.BlockOwnerTxIndex.Put(tx.Link[:], blockNumberKey)
					storage.Pool.BlockOwnerTxIndex.Delete(txId[:])

				case *transactionrecord.BlockOwnerTransfer:
					err := transactionrecord.CheckPayments(prevTx.Version, mode.IsTesting(), prevTx.Payments)
					if nil != err {
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					packedPayments, err := prevTx.Payments.Pack(mode.IsTesting())
					if nil != err {
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					// payment data
					storage.Pool.BlockOwnerPayment.Put(blockNumberKey, packedPayments)
					storage.Pool.BlockOwnerTxIndex.Put(tx.Link[:], blockNumberKey)
					storage.Pool.BlockOwnerTxIndex.Delete(txId[:])

				default:
					logger.Criticalf("invalid block transfer link: %+v", prevTx)
					logger.Panic("Transaction database is corrupt")
				}

			case *transactionrecord.BitmarkShare:
				txId := packedTransaction.MakeLink()
				blockNumber, linkOwner := ownership.OwnerOf(tx.Link)
				if nil == linkOwner {
					log.Criticalf("missing transaction record for: %v", tx.Link)
					logger.Panic("Transactions database is corrupt")
				}

				ownerData, err := ownership.GetOwnerData(txId)
				if nil != err {
					logger.Criticalf("invalid ownerData for tx id: %s", txId)
					logger.Panic("Ownership database is corrupt")
				}
				shareData, ok := ownerData.(*ownership.ShareOwnerData)
				if !ok {
					logger.Criticalf("invalid ownerData: %+v for tx id: %s", ownerData, txId)
					logger.Panic("Ownership database is corrupt")
				}

				storage.Pool.Transactions.Delete(txId[:])
				reservoir.DeleteByTxId(txId)

				shareId := shareData.IssueTxId()

				fKey := append(linkOwner.Bytes(), shareId[:]...)
				storage.Pool.Shares.Delete(shareId[:])
				storage.Pool.ShareQuantity.Delete(fKey)

				ownership.Transfer(txId, tx.Link, blockNumber, linkOwner, linkOwner)

			case *transactionrecord.ShareGrant:

				txId := packedTransaction.MakeLink()

				storage.Pool.Transactions.Delete(txId[:])
				reservoir.DeleteByTxId(txId)

				oKey := append(tx.Owner.Bytes(), tx.ShareId[:]...)
				rKey := append(tx.Recipient.Bytes(), tx.ShareId[:]...)

				// this could be zero
				oAccountBalance, _ := storage.Pool.ShareQuantity.GetN(oKey)

				// this cannot be zero
				rAccountBalance, ok := storage.Pool.ShareQuantity.GetN(rKey)
				if !ok {
					log.Criticalf("missing balance record for: %v share id: %x", tx.Recipient, tx.ShareId)
					logger.Panic("ShareQuantity database is corrupt")
				}

				// owner, share ← recipient
				rAccountBalance -= tx.Quantity
				oAccountBalance += tx.Quantity

				// update balances
				if 0 == rAccountBalance {
					storage.Pool.ShareQuantity.Delete(rKey)
				} else {
					storage.Pool.ShareQuantity.PutN(rKey, rAccountBalance)
				}
				storage.Pool.ShareQuantity.PutN(oKey, oAccountBalance)

			case *transactionrecord.ShareSwap:

				txId := packedTransaction.MakeLink()

				storage.Pool.Transactions.Delete(txId[:])
				reservoir.DeleteByTxId(txId)

				ownerOneShareOneKey := append(tx.OwnerOne.Bytes(), tx.ShareIdOne[:]...)
				ownerOneShareTwoKey := append(tx.OwnerOne.Bytes(), tx.ShareIdTwo[:]...)
				ownerTwoShareOneKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdOne[:]...)
				ownerTwoShareTwoKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdTwo[:]...)

				// either of these balances could be zero
				ownerOneShareOneAccountBalance, _ := storage.Pool.ShareQuantity.GetN(ownerOneShareOneKey)
				ownerTwoShareTwoAccountBalance, _ := storage.Pool.ShareQuantity.GetN(ownerTwoShareTwoKey)

				// these balances cannot be zero
				ownerOneShareTwoAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerOneShareTwoKey)
				if !ok {
					log.Criticalf("missing balance record for owner 1: %v share id 2: %x", tx.OwnerOne, tx.ShareIdTwo)
					logger.Panic("ShareQuantity database is corrupt")
				}
				ownerTwoShareOneAccountBalance, ok := storage.Pool.ShareQuantity.GetN(ownerTwoShareOneKey)
				if !ok {
					log.Criticalf("missing balance record for owner 2: %v share id 1: %x", tx.OwnerTwo, tx.ShareIdOne)
					logger.Panic("ShareQuantity database is corrupt")
				}

				// owner 1, share 1 ← owner 2
				ownerTwoShareOneAccountBalance -= tx.QuantityOne
				ownerOneShareOneAccountBalance += tx.QuantityOne

				// owner 2, share 2 ← owner 1
				ownerOneShareTwoAccountBalance -= tx.QuantityTwo
				ownerTwoShareTwoAccountBalance += tx.QuantityTwo

				// update database share one
				if 0 == ownerTwoShareOneAccountBalance {
					storage.Pool.ShareQuantity.Delete(ownerTwoShareOneKey)
				} else {
					storage.Pool.ShareQuantity.PutN(ownerTwoShareOneKey, ownerTwoShareOneAccountBalance)
				}
				storage.Pool.ShareQuantity.PutN(ownerOneShareOneKey, ownerOneShareOneAccountBalance)

				// update database share two
				if 0 == ownerOneShareTwoAccountBalance {
					storage.Pool.ShareQuantity.Delete(ownerOneShareTwoKey)
				} else {
					storage.Pool.ShareQuantity.PutN(ownerOneShareTwoKey, ownerOneShareTwoAccountBalance)
				}
				storage.Pool.ShareQuantity.PutN(ownerTwoShareTwoKey, ownerTwoShareTwoAccountBalance)

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
		storage.Pool.BlockOwnerPayment.Delete(blockNumberKey)
		storage.Pool.Blocks.Delete(blockNumberKey)

		// and delete its hash
		storage.Pool.BlockHeaderHash.Delete(blockNumberKey)

		// fetch previous block number
		binary.BigEndian.PutUint64(blockNumberKey, header.Number-1)
		packedBlock = storage.Pool.Blocks.Get(blockNumberKey)

		if nil == packedBlock {
			// all blocks deleted
			blockheader.SetGenesis()
			break outer_loop
		}

	}
	return nil
}
