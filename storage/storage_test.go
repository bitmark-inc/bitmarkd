// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage_test

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/transactionrecord"

	"github.com/bitmark-inc/bitmarkd/storage"
)

const (
	number = 1234

	assetID             = 5
	blockHeaderID       = 2
	blockID             = 7
	blockOwnerTXID      = 3
	blockOwnerPaymentID = 6
	ownerDataID         = 10
	ownerListID         = 1
	ownerTXID           = 9
	ownerNextCountID    = 4
	shareQuantityID     = 12
	sharesID            = 11
	txID                = 8
)

var (
	packedAsset             = transactionrecord.Packed{assetID}
	packedBlockHeader       = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, blockHeaderID}
	packedBlock             = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, blockID}
	packedBlockOwnerTX      = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, blockOwnerTXID}
	packedBlockOwnerPayment = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, blockOwnerPaymentID}
	packedOwnerData         = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, ownerDataID}
	packedOwnerList         = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, ownerListID}
	packedOwnerTXID         = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, ownerTXID}
	packedOwnerNextCount    = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, ownerNextCountID}
	packedShareQuantity     = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, shareQuantityID}
	packedShares            = transactionrecord.Packed{0, 0, 0, 0, 0, 0, 0, 0, sharesID}
	packedTransaction       = transactionrecord.Packed{txID}

	blockNumber []byte
)

func init() {
	blockNumber = make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumber, uint64(number))
	packedTransaction = transactionrecord.Packed{txID}
	packedAsset = transactionrecord.Packed{assetID}
}

func setupTransaction() storage.Transaction {
	trx, _ := storage.NewDBTransaction()
	return trx
}

func TestAssetsPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.Assets
	trx.Put(pool, []byte{assetID}, blockNumber, packedAsset)
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{assetID})

	tempData := make([]byte, 8)
	copy(tempData, blockNumber)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong asset data")
	assert.Equal(t, []byte{assetID}, key, "wrong asset key")
}

func TestTransactionsPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.Transactions
	trx.Put(pool, []byte{txID}, blockNumber, packedTransaction)
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{txID})

	tempData := make([]byte, 8)
	copy(tempData, blockNumber)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong transaction data")
	assert.Equal(t, []byte{txID}, key, "wrong transaction key")
}

func TestBlockPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.Blocks
	trx.Put(pool, []byte{blockID}, packedBlock, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{blockID})

	tempData := make([]byte, 9)
	copy(tempData, packedBlock)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong block data")
	assert.Equal(t, []byte{blockID}, key, "wrong block key")
}

func TestBlockHeaderHashPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.BlockHeaderHash
	trx.Put(pool, []byte{blockHeaderID}, packedBlockHeader, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{blockHeaderID})

	tempData := make([]byte, 9)
	copy(tempData, packedBlockHeader)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong block header data")
	assert.Equal(t, []byte{blockHeaderID}, key, "wrong block header key")
}

func TestBlockOwnerPaymentPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.BlockOwnerPayment
	trx.Put(pool, []byte{blockOwnerPaymentID}, packedBlockOwnerPayment, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{blockOwnerPaymentID})

	tempData := make([]byte, 9)
	copy(tempData, packedBlockOwnerPayment)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong block owner payment data")
	assert.Equal(t, []byte{blockOwnerPaymentID}, key, "wrong block owner payment key")
}

func TestBlockOwnerTXPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.BlockOwnerPayment
	trx.Put(pool, []byte{blockOwnerTXID}, packedBlockOwnerTX, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{blockOwnerTXID})

	tempData := make([]byte, 9)
	copy(tempData, packedBlockOwnerTX)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong block owner tx data")
	assert.Equal(t, []byte{blockOwnerTXID}, key, "wrong block owner tx key")
}

func TestOwnerNextCountPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.OwnerNextCount
	trx.Put(pool, []byte{ownerNextCountID}, packedOwnerNextCount, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{ownerNextCountID})

	tempData := make([]byte, 9)
	copy(tempData, packedOwnerNextCount)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong owner next count data")
	assert.Equal(t, []byte{ownerNextCountID}, key, "wrong owner next count key")
}

func TestOwnerListPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.OwnerList
	trx.Put(pool, []byte{ownerListID}, packedOwnerList, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{ownerListID})

	tempData := make([]byte, 9)
	copy(tempData, packedOwnerList)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong owner list data")
	assert.Equal(t, []byte{ownerListID}, key, "wrong owner list key")
}

func TestOwnerTXIDPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.OwnerTxIndex
	trx.Put(pool, []byte{ownerTXID}, packedOwnerTXID, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{ownerTXID})

	tempData := make([]byte, 9)
	copy(tempData, packedOwnerTXID)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong owner tx id data")
	assert.Equal(t, []byte{ownerTXID}, key, "wrong owner tx id key")
}

func TestOwnerDataPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.OwnerData
	trx.Put(pool, []byte{ownerDataID}, packedOwnerData, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{ownerDataID})

	tempData := make([]byte, 9)
	copy(tempData, packedOwnerData)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong owner data")
	assert.Equal(t, []byte{ownerDataID}, key, "wrong owner data key")
}

func TestSharePut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.Shares
	trx.Put(pool, []byte{sharesID}, packedShares, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{sharesID})

	tempData := make([]byte, 9)
	copy(tempData, packedShares)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong share data")
	assert.Equal(t, []byte{sharesID}, key, "wrong share key")
}

func TestShareQuantityPut(t *testing.T) {
	trx := setupTransaction()
	pool := storage.Pool.ShareQuantity
	trx.Put(pool, []byte{shareQuantityID}, packedShareQuantity, []byte{})
	_ = trx.Commit()

	data, key := trx.GetNB(pool, []byte{shareQuantityID})

	tempData := make([]byte, 9)
	copy(tempData, packedShareQuantity)
	expected := binary.BigEndian.Uint64(tempData[:])

	assert.Equal(t, expected, data, "wrong share quantity data")
	assert.Equal(t, []byte{shareQuantityID}, key, "wrong share quantity key")
}
