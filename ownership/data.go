// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ownership

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// from storage/doc.go:
//
// Ownership:
//
//   N ⧺ owner            - next count value to use for appending to owned items
//                          data: count
//   L ⧺ owner ⧺ count    - list of owned items
//                          data: txId
//   D ⧺ owner ⧺ txId     - position in list of owned items, for delete after transfer
//                          data: count
//   P ⧺ txId             - owner data (00=asset, 01=block, 02=share) head of provenance chain
//                          data: 00 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ asset id
//                          data: 01 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ owned BN
//                          data: 02 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ asset id
//
//
// Bitmark Shares (txId ≡ share id)
//
//   F ⧺ txId             - share total value (constant)
//                          data: value
//   Q ⧺ owner ⧺ txId     - current balance quantity of shares (ShareId) for each owner (deleted if value becomes zero)

const (
	oneByteSize    = 1
	uint64ByteSize = 8
)

// structure of the ownership record
const (
	itemStart  = 0
	itemFinish = itemStart + oneByteSize

	transferBlockNumberStart  = itemFinish
	transferBlockNumberFinish = transferBlockNumberStart + uint64ByteSize

	issueTxIdStart  = transferBlockNumberFinish
	issueTxIdFinish = issueTxIdStart + merkle.DigestLength

	issueBlockNumberStart  = issueTxIdFinish
	issueBlockNumberFinish = issueBlockNumberStart + uint64ByteSize

	// Overlap OwnedAsset/OwnedShare
	assetIdentifierStart  = issueBlockNumberFinish
	assetIdentifierFinish = assetIdentifierStart + transactionrecord.AssetIdentifierLength

	// Overlap OwnedBlock
	//ownedBlockNumberStart  = issueBlockNumberFinish
	//ownedBlockNumberFinish = ownedBlockNumberStart + uint64ByteSize

	// length of the packed items
	assetPackLength = assetIdentifierFinish
	blockPackLength = issueBlockNumberFinish
	sharePackLength = assetIdentifierFinish
)

// AssetOwnerData - owner data
type AssetOwnerData struct {
	transferBlockNumber uint64
	issueTxId           merkle.Digest
	issueBlockNumber    uint64
	assetId             transactionrecord.AssetIdentifier
}

// BlockOwnerData - owner data
type BlockOwnerData struct {
	transferBlockNumber uint64
	issueTxId           merkle.Digest
	issueBlockNumber    uint64 // also this is the number of the owned block
}

// ShareOwnerData - owner data
type ShareOwnerData struct {
	transferBlockNumber uint64
	issueTxId           merkle.Digest
	issueBlockNumber    uint64
	assetId             transactionrecord.AssetIdentifier
}

// OwnerData - generic owner data methods
type OwnerData interface {
	Pack() PackedOwnerData

	IssueTxId() merkle.Digest
	TransferBlockNumber() uint64
	IssueBlockNumber() uint64
}

// PackedOwnerData - packed data to store in database
type PackedOwnerData []byte

// GetOwnerData - fetch and unpack owner data
func GetOwnerData(trx storage.Transaction, txId merkle.Digest, ownerDataHandle storage.Handle) (OwnerData, error) {
	var packed []byte
	if nil == trx {
		packed = ownerDataHandle.Get(txId[:])
	} else {
		packed = trx.Get(ownerDataHandle, txId[:])
	}

	if nil == packed {
		return nil, fault.MissingOwnerData
	}

	return PackedOwnerData(packed).Unpack()
}

// GetOwnerDataB - fetch and unpack owner data
func GetOwnerDataB(trx storage.Transaction, txId []byte, ownerDataHandle storage.Handle) (OwnerData, error) {
	var packed []byte
	if nil == trx {
		packed = ownerDataHandle.Get(txId)
	} else {
		packed = trx.Get(ownerDataHandle, txId)
	}

	if nil == packed {
		return nil, fault.MissingOwnerData
	}

	return PackedOwnerData(packed).Unpack()
}

// Pack - pack asset owner data to byte slice
func (a AssetOwnerData) Pack() PackedOwnerData {

	// 8 byte block number
	trBN := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(trBN, a.transferBlockNumber)

	isBN := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(isBN, a.issueBlockNumber)

	newData := make(PackedOwnerData, 0, assetPackLength)

	newData = append(newData, byte(OwnedAsset))
	newData = append(newData, trBN...)
	newData = append(newData, a.issueTxId[:]...)
	newData = append(newData, isBN...)
	newData = append(newData, a.assetId[:]...)

	return newData
}

// accessors
func (a AssetOwnerData) IssueTxId() merkle.Digest {
	return a.issueTxId
}
func (a AssetOwnerData) TransferBlockNumber() uint64 {
	return a.transferBlockNumber
}
func (a AssetOwnerData) IssueBlockNumber() uint64 {
	return a.issueBlockNumber
}

// Pack - pack block owner data to byte slice
func (b BlockOwnerData) Pack() PackedOwnerData {

	// 8 byte block number
	trBN := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(trBN, b.transferBlockNumber)

	isBN := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(isBN, b.issueBlockNumber)

	newData := make(PackedOwnerData, 0, blockPackLength)

	newData = append(newData, byte(OwnedBlock))
	newData = append(newData, trBN...)
	newData = append(newData, b.issueTxId[:]...)
	newData = append(newData, isBN...)

	return newData
}

// accessors
func (b BlockOwnerData) IssueTxId() merkle.Digest {
	return b.issueTxId
}
func (b BlockOwnerData) TransferBlockNumber() uint64 {
	return b.transferBlockNumber
}
func (b BlockOwnerData) IssueBlockNumber() uint64 {
	return b.issueBlockNumber
}

// Pack - pack share owner data to byte slice
func (a ShareOwnerData) Pack() PackedOwnerData {

	// 8 byte block number
	trBN := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(trBN, a.transferBlockNumber)

	isBN := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(isBN, a.issueBlockNumber)

	newData := make(PackedOwnerData, 0, sharePackLength)

	newData = append(newData, byte(OwnedShare))
	newData = append(newData, trBN...)
	newData = append(newData, a.issueTxId[:]...)
	newData = append(newData, isBN...)
	newData = append(newData, a.assetId[:]...)

	return newData
}

// accessors
func (a ShareOwnerData) IssueTxId() merkle.Digest {
	return a.issueTxId
}
func (a ShareOwnerData) TransferBlockNumber() uint64 {
	return a.transferBlockNumber
}
func (a ShareOwnerData) IssueBlockNumber() uint64 {
	return a.issueBlockNumber
}

// Unpack - unpack record into the appropriate type
func (packed PackedOwnerData) Unpack() (OwnerData, error) {
	if len(packed) < 1 {
		return nil, fault.NotOwnerDataPack
	}
	switch OwnedItem(packed[itemStart]) {

	case OwnedAsset:
		if assetPackLength != len(packed) {
			return nil, fault.NotOwnerDataPack
		}
		a := &AssetOwnerData{
			transferBlockNumber: binary.BigEndian.Uint64(packed[transferBlockNumberStart:transferBlockNumberFinish]),
			issueBlockNumber:    binary.BigEndian.Uint64(packed[issueBlockNumberStart:issueBlockNumberFinish]),
		}
		merkle.DigestFromBytes(&a.issueTxId, packed[issueTxIdStart:issueTxIdFinish])
		transactionrecord.AssetIdentifierFromBytes(&a.assetId, packed[assetIdentifierStart:assetIdentifierFinish])

		return a, nil

	case OwnedBlock:
		if blockPackLength != len(packed) {
			return nil, fault.NotOwnerDataPack
		}
		b := &BlockOwnerData{
			transferBlockNumber: binary.BigEndian.Uint64(packed[transferBlockNumberStart:transferBlockNumberFinish]),
			issueBlockNumber:    binary.BigEndian.Uint64(packed[issueBlockNumberStart:issueBlockNumberFinish]),
		}
		merkle.DigestFromBytes(&b.issueTxId, packed[issueTxIdStart:issueTxIdFinish])
		return b, nil

	case OwnedShare:
		if sharePackLength != len(packed) {
			return nil, fault.NotOwnerDataPack
		}
		a := &ShareOwnerData{
			transferBlockNumber: binary.BigEndian.Uint64(packed[transferBlockNumberStart:transferBlockNumberFinish]),
			issueBlockNumber:    binary.BigEndian.Uint64(packed[issueBlockNumberStart:issueBlockNumberFinish]),
		}
		merkle.DigestFromBytes(&a.issueTxId, packed[issueTxIdStart:issueTxIdFinish])
		transactionrecord.AssetIdentifierFromBytes(&a.assetId, packed[assetIdentifierStart:assetIdentifierFinish])
		return a, nil

	default:
		return nil, fault.NotOwnerDataPack
	}
}
