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
			// start db transaction by block & index db
			trx, err := storage.NewDBTransaction()
			if nil != err {
				log.Errorf("cannot create transaction: error: %s", i, err)
				return err
			}

			transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
			if nil != err {
				trx.Abort()
				log.Warnf("invalid tx[%d]: error: %s", i, err)
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
				trx.Delete(storage.Pool.Assets, assetId[:])
				asset.Delete(assetId)

			case *transactionrecord.BitmarkIssue:
				txId := packedTransaction.MakeLink()
				reservoir.DeleteByTxId(txId)
				if storage.Pool.Transactions.Has(txId[:]) {
					trx.Delete(storage.Pool.Transactions, txId[:])
					ownership.Transfer(trx, txId, txId, 0, tx.Owner, nil)
				}

			case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
				tr := tx.(transactionrecord.BitmarkTransfer)
				txId := packedTransaction.MakeLink()
				trx.Delete(storage.Pool.Transactions, txId[:])
				reservoir.DeleteByTxId(txId)
				link := tr.GetLink()
				blockNumber, linkOwner := ownership.OwnerOf(trx, link)
				if nil == linkOwner {
					trx.Abort()
					log.Criticalf("missing transaction record for: %v", link)
					logger.Panic("Transactions database is corrupt")
				}
				ownership.Transfer(trx, txId, link, blockNumber, tr.GetOwner(), linkOwner)

			case *transactionrecord.BlockFoundation:
				if nil == blockOwner {
					blockOwner = tx.Owner
				}
				// delete later

			case *transactionrecord.BlockOwnerTransfer:
				txId := packedTransaction.MakeLink()
				trx.Delete(storage.Pool.Transactions, txId[:])
				reservoir.DeleteByTxId(txId)
				blockNumber, linkOwner := ownership.OwnerOf(trx, tx.Link)
				if nil == linkOwner {
					trx.Abort()
					log.Criticalf("missing transaction record for: %v", tx.Link)
					logger.Panic("Transactions database is corrupt")
				}
				ownerdata, err := ownership.GetOwnerDataB(trx, txId[:])
				if nil != err {
					trx.Abort()
					log.Criticalf("missing ownership for: %s", txId)
					logger.Panic("Ownership database is corrupt")
				}
				blockOwnerdata, ok := ownerdata.(*ownership.BlockOwnerData)
				if !ok {
					trx.Abort()
					log.Criticalf("expected block ownership but read: %+v", ownerdata)
					logger.Panic("Ownership database is corrupt")
				}

				ownership.Transfer(trx, txId, tx.Link, blockNumber, tx.Owner, linkOwner)

				blockNumberKey := make([]byte, 8)
				binary.BigEndian.PutUint64(blockNumberKey, blockOwnerdata.IssueBlockNumber())

				// put block ownership back
				_, previous := trx.GetNB(storage.Pool.Transactions, tx.Link[:])

				blockTransaction, _, err := transactionrecord.Packed(previous).Unpack(mode.IsTesting())
				if nil != err {
					trx.Abort()
					logger.Criticalf("invalid error: %s", txId, err)
					logger.Panic("Transaction database is corrupt")
				}
				switch prevTx := blockTransaction.(type) {
				case *transactionrecord.BlockFoundation:
					err := transactionrecord.CheckPayments(prevTx.Version, mode.IsTesting(), prevTx.Payments)
					if nil != err {
						trx.Abort()
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					packedPayments, err := prevTx.Payments.Pack(mode.IsTesting())
					if nil != err {
						trx.Abort()
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					// payment data
					trx.Put(
						storage.Pool.BlockOwnerPayment,
						blockNumberKey,
						packedPayments,
						[]byte{},
					)
					trx.Put(
						storage.Pool.BlockOwnerTxIndex,
						tx.Link[:],
						blockNumberKey,
						[]byte{},
					)
					trx.Delete(storage.Pool.BlockOwnerTxIndex, txId[:])

				case *transactionrecord.BlockOwnerTransfer:
					err := transactionrecord.CheckPayments(prevTx.Version, mode.IsTesting(), prevTx.Payments)
					if nil != err {
						trx.Abort()
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					packedPayments, err := prevTx.Payments.Pack(mode.IsTesting())
					if nil != err {
						trx.Abort()
						logger.Criticalf("invalid tx id: %s  error: %s", txId, err)
						logger.Panic("Transaction database is corrupt")
					}
					// payment data
					trx.Put(storage.Pool.BlockOwnerPayment, blockNumberKey, packedPayments, []byte{})
					trx.Put(storage.Pool.BlockOwnerTxIndex, tx.Link[:], blockNumberKey, []byte{})
					trx.Delete(storage.Pool.BlockOwnerTxIndex, txId[:])

				default:
					trx.Abort()
					logger.Criticalf("invalid block transfer link: %+v", prevTx)
					logger.Panic("Transaction database is corrupt")
				}

			case *transactionrecord.BitmarkShare:
				txId := packedTransaction.MakeLink()
				blockNumber, linkOwner := ownership.OwnerOf(trx, tx.Link)
				if nil == linkOwner {
					trx.Abort()
					log.Criticalf("missing transaction record for: %v", tx.Link)
					logger.Panic("Transactions database is corrupt")
				}

				ownerData, err := ownership.GetOwnerData(trx, txId)
				if nil != err {
					trx.Abort()
					logger.Criticalf("invalid ownerData for tx id: %s", txId)
					logger.Panic("Ownership database is corrupt")
				}
				shareData, ok := ownerData.(*ownership.ShareOwnerData)
				if !ok {
					trx.Abort()
					logger.Criticalf("invalid ownerData: %+v for tx id: %s", ownerData, txId)
					logger.Panic("Ownership database is corrupt")
				}

				trx.Delete(storage.Pool.Transactions, txId[:])
				reservoir.DeleteByTxId(txId)

				shareId := shareData.IssueTxId()

				fKey := append(linkOwner.Bytes(), shareId[:]...)
				trx.Delete(storage.Pool.Shares, shareId[:])
				trx.Delete(storage.Pool.ShareQuantity, fKey)

				ownership.Transfer(trx, txId, tx.Link, blockNumber, linkOwner, linkOwner)

			case *transactionrecord.ShareGrant:

				txId := packedTransaction.MakeLink()

				trx.Delete(storage.Pool.Transactions, txId[:])
				reservoir.DeleteByTxId(txId)

				oKey := append(tx.Owner.Bytes(), tx.ShareId[:]...)
				rKey := append(tx.Recipient.Bytes(), tx.ShareId[:]...)

				// this could be zero
				oAccountBalance, _ := trx.GetN(storage.Pool.ShareQuantity, oKey)

				// this cannot be zero
				rAccountBalance, ok := trx.GetN(storage.Pool.ShareQuantity, rKey)
				if !ok {
					trx.Abort()
					log.Criticalf("missing balance record for: %v share id: %x", tx.Recipient, tx.ShareId)
					logger.Panic("ShareQuantity database is corrupt")
				}

				// owner, share ← recipient
				rAccountBalance -= tx.Quantity
				oAccountBalance += tx.Quantity

				// update balances
				if 0 == rAccountBalance {
					trx.Delete(storage.Pool.ShareQuantity, rKey)
				} else {
					trx.PutN(storage.Pool.ShareQuantity, rKey, rAccountBalance)
				}
				trx.PutN(storage.Pool.ShareQuantity, oKey, oAccountBalance)

			case *transactionrecord.ShareSwap:

				txId := packedTransaction.MakeLink()

				trx.Delete(storage.Pool.Transactions, txId[:])
				reservoir.DeleteByTxId(txId)

				ownerOneShareOneKey := append(tx.OwnerOne.Bytes(), tx.ShareIdOne[:]...)
				ownerOneShareTwoKey := append(tx.OwnerOne.Bytes(), tx.ShareIdTwo[:]...)
				ownerTwoShareOneKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdOne[:]...)
				ownerTwoShareTwoKey := append(tx.OwnerTwo.Bytes(), tx.ShareIdTwo[:]...)

				// either of these balances could be zero
				ownerOneShareOneAccountBalance, _ := trx.GetN(storage.Pool.ShareQuantity, ownerOneShareOneKey)
				ownerTwoShareTwoAccountBalance, _ := trx.GetN(storage.Pool.ShareQuantity, ownerTwoShareTwoKey)

				// these balances cannot be zero
				ownerOneShareTwoAccountBalance, ok := trx.GetN(storage.Pool.ShareQuantity, ownerOneShareTwoKey)
				if !ok {
					trx.Abort()
					log.Criticalf("missing balance record for owner 1: %v share id 2: %x", tx.OwnerOne, tx.ShareIdTwo)
					logger.Panic("ShareQuantity database is corrupt")
				}
				ownerTwoShareOneAccountBalance, ok := trx.GetN(storage.Pool.ShareQuantity, ownerTwoShareOneKey)
				if !ok {
					trx.Abort()
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
					trx.Delete(storage.Pool.ShareQuantity, ownerTwoShareOneKey)
				} else {
					trx.PutN(storage.Pool.ShareQuantity, ownerTwoShareOneKey, ownerTwoShareOneAccountBalance)
				}
				trx.PutN(storage.Pool.ShareQuantity, ownerOneShareOneKey, ownerOneShareOneAccountBalance)

				// update database share two
				if 0 == ownerOneShareTwoAccountBalance {
					trx.Delete(storage.Pool.ShareQuantity, ownerOneShareTwoKey)
				} else {
					trx.PutN(storage.Pool.ShareQuantity, ownerOneShareTwoKey, ownerOneShareTwoAccountBalance)
				}
				trx.PutN(storage.Pool.ShareQuantity, ownerTwoShareTwoKey, ownerTwoShareTwoAccountBalance)

			default:
				trx.Abort()
				logger.Panicf("unexpected transaction: %v", transaction)
			}

			data = data[n:]
			if 0 == len(data) {
				trx.Commit()
				break inner_loop
			}

			// commit db transactions
			trx.Commit()
		}

		// start db transaction by block & index db
		trx, err := storage.NewDBTransaction()
		if nil != err {
			return err
		}

		// block number key for deletion
		blockNumberKey := make([]byte, 8)
		binary.BigEndian.PutUint64(blockNumberKey, header.Number)

		// block ownership remove
		foundationTxId := blockrecord.FoundationTxId(header, digest)
		trx.Delete(storage.Pool.Transactions, foundationTxId[:])
		if nil == blockOwner {
			trx.Abort()
			log.Criticalf("nil block owner for block: %d", header.Number)
		} else {
			ownership.Transfer(trx, foundationTxId, foundationTxId, 0, blockOwner, nil)
		}
		// remove remaining block data
		trx.Delete(storage.Pool.BlockOwnerTxIndex, foundationTxId[:])
		trx.Delete(storage.Pool.BlockOwnerPayment, blockNumberKey)
		trx.Delete(storage.Pool.Blocks, blockNumberKey)

		// and delete its hash
		trx.Delete(storage.Pool.BlockHeaderHash, blockNumberKey)

		// fetch previous block number
		binary.BigEndian.PutUint64(blockNumberKey, header.Number-1)
		packedBlock = storage.Pool.Blocks.Get(blockNumberKey)

		if nil == packedBlock {
			// all blocks deleted
			blockheader.SetGenesis()
			trx.Commit()
			break outer_loop
		}

		// commit db transactions
		trx.Commit()
	}
	return nil
}
