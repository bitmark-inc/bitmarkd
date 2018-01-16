// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/currency/bitcoin"
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

	// this sets the maximum number of currencies supported
	// order is determined by the currency enum order
	currencyAddresses := [currency.Count]string{} // bitcoin, litecoin

	// store transactions
	for i, item := range txs {
		txId := txIds[i]
		packed := item.packed
		switch tx := item.unpacked.(type) {

		case *transactionrecord.BaseData:
			// ensure base data is at the start of a block
			if i >= len(currencyAddresses) {
				return fault.ErrOutOfPlaceBaseData
			}

			// ensure order follows currency enum order
			if tx.Currency.Index() != i {
				return fault.ErrOutOfPlaceBaseData
			}

			// extract the currency address for payments
			switch tx.Currency {
			case currency.Bitcoin:
				cType, _, err := bitcoin.ValidateAddress(tx.PaymentAddress)
				if nil != err {
					return err
				}
				switch cType {
				case bitcoin.Testnet, bitcoin.TestnetScript:
					if !mode.IsTesting() {
						return fault.ErrBitcoinAddressForWrongNetwork
					}
				case bitcoin.Livenet, bitcoin.LivenetScript:
					if mode.IsTesting() {
						return fault.ErrBitcoinAddressForWrongNetwork
					}
				default:
					return fault.ErrBitcoinAddressIsNotSupported
				}
				// save bitcoin address
				currencyAddresses[0] = tx.PaymentAddress

				// simulate a litecoin address (from this bitcoin address) as a default
				// and to provide a litecoin address for older blocks with no litecoin base record
				currencyAddresses[1], err = litecoin.FromBitcoin(tx.PaymentAddress)

			case currency.Litecoin:
				cType, _, err := litecoin.ValidateAddress(tx.PaymentAddress)
				if nil != err {
					return err
				}
				switch cType {
				case litecoin.Testnet, litecoin.TestnetScript:
					if !mode.IsTesting() {
						return fault.ErrLitecoinAddressForWrongNetwork
					}
				case litecoin.Livenet, litecoin.LivenetScript, litecoin.LivenetScript2:
					if mode.IsTesting() {
						return fault.ErrLitecoinAddressForWrongNetwork
					}
				default:
					return fault.ErrLitecoinAddressIsNotSupported
				}
				// save litecoin address
				currencyAddresses[1] = tx.PaymentAddress

			default:
				return fault.ErrInvalidCurrency
			}

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

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			tr := tx.(transactionrecord.BitmarkTransfer)
			key := txId[:]
			reservoir.DeleteByTxId(txId)
			link := tr.GetLink()

			// when deleting a pending it is possible that the tx id
			// it was holding was different to this tx id
			// i.e. it is a duplicate so it also must be removed
			// to prevent the possibility of a double-spend
			reservoir.DeleteByLink(link)

			storage.Pool.Transactions.Put(key, packed)
			linkOwner := OwnerOf(link)
			if nil == linkOwner {
				logger.Criticalf("missing transaction record for link: %v refererenced by tx id: %v", link, txId)
				logger.Panic("Transactions database is corrupt")
			}
			TransferOwnership(link, txId, header.Number, linkOwner, tr.GetOwner())

		default:
			globalData.log.Criticalf("unhandled transaction: %v", tx)
			logger.Panicf("unhandled transaction: %v", tx)
		}
	}

	// currency database write
	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	byteCount := 0
	for _, s := range currencyAddresses {
		byteCount += len(s) + 1 // include a 0x00 separator byte as each string is Base58 ASCII text
	}
	currencyData := make([]byte, 0, byteCount)
	for _, s := range currencyAddresses {
		currencyData = append(currencyData, s...)
		currencyData = append(currencyData, 0x00)
	}
	storage.Pool.BlockOwners.Put(blockNumber, currencyData)

	// finish be stoing the block header
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

	blockNumber := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, header.Number)

	storage.Pool.Blocks.Put(blockNumber, packedBlock)
}
