// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ownership

import (
	"bytes"
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Record - type to represent an ownership record
type Record struct {
	N           uint64                             `json:"n,string"`
	TxId        merkle.Digest                      `json:"txId"`
	IssueTxId   merkle.Digest                      `json:"issue"`
	Item        OwnedItem                          `json:"item"`
	AssetId     *transactionrecord.AssetIdentifier `json:"assetId,omitempty"`
	BlockNumber *uint64                            `json:"blockNumber,omitempty"`
}

// listBitmarksFor - fetch a list of bitmarks for an owner
func listBitmarksFor(owner *account.Account, start uint64, count int) ([]Record, error) {

	startBytes := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(startBytes, start)

	ownerBytes := owner.Bytes()
	prefix := append([]byte{}, ownerBytes...)
	prefix = append(prefix, startBytes...)

	cursor := storage.Pool.OwnerList.NewFetchCursor().Seek(prefix)

	// owner ⧺ count → txId
	items, err := cursor.Fetch(count)
	if err != nil {
		return nil, err
	}

	records := make([]Record, 0, len(items))

loop:
	for _, item := range items {
		n := len(item.Key)
		split := n - uint64ByteSize
		if split <= 0 {
			logger.Panicf("split cannot be <= 0: %d", split)
		}
		itemOwner := item.Key[:n-uint64ByteSize]
		if !bytes.Equal(ownerBytes, itemOwner) {
			break loop
		}

		record := Record{
			N: binary.BigEndian.Uint64(item.Key[split:]),
		}

		merkle.DigestFromBytes(&record.TxId, item.Value)

		ownerData, err := GetOwnerData(nil, record.TxId, storage.Pool.OwnerData)
		if err != nil {
			return nil, err
		}

		switch od := ownerData.(type) {
		case *AssetOwnerData:
			record.Item = OwnedAsset
			record.IssueTxId = od.issueTxId
			record.AssetId = &od.assetId

		case *BlockOwnerData:
			record.Item = OwnedBlock
			record.IssueTxId = od.issueTxId
			record.BlockNumber = &od.issueBlockNumber

		case *ShareOwnerData:
			record.Item = OwnedShare
			record.IssueTxId = od.issueTxId
			record.AssetId = &od.assetId

		default:
			logger.Panicf("unsupported item type: %d", item)
		}
		records = append(records, record)
	}

	return records, nil
}
