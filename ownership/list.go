// Copyright (c) 2014-2018 Bitmark Inc.
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

// type to represent an ownership record
type Ownership struct {
	N           uint64                             `json:"n,string"`
	TxId        merkle.Digest                      `json:"txId"`
	IssueTxId   merkle.Digest                      `json:"issue"`
	Item        OwnedItem                          `json:"item"`
	AssetId     *transactionrecord.AssetIdentifier `json:"assetId,omitempty"`
	BlockNumber *uint64                            `json:"blockNumber,omitempty"`
}

// fetch a list of bitmarks for an owner
func ListBitmarksFor(owner *account.Account, start uint64, count int) ([]Ownership, error) {

	startBytes := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(startBytes, start)

	ownerBytes := owner.Bytes()
	prefix := append(ownerBytes, startBytes...)

	cursor := storage.Pool.Ownership.NewFetchCursor().Seek(prefix)

	items, err := cursor.Fetch(count)
	if nil != err {
		return nil, err
	}

	records := make([]Ownership, 0, len(items))

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

		record := Ownership{
			N: binary.BigEndian.Uint64(item.Key[split:]),
		}

		merkle.DigestFromBytes(&record.TxId, item.Value[TxIdStart:TxIdFinish])
		merkle.DigestFromBytes(&record.IssueTxId, item.Value[IssueTxIdStart:IssueTxIdFinish])

		switch itemType := OwnedItem(item.Value[FlagByteStart]); itemType {
		case OwnedAsset:
			a := &transactionrecord.AssetIdentifier{}
			transactionrecord.AssetIdentifierFromBytes(a, item.Value[AssetIdentifierStart:AssetIdentifierFinish])
			record.AssetId = a
			record.Item = itemType
		case OwnedBlock:
			b := binary.BigEndian.Uint64(item.Value[OwnedBlockNumberStart:OwnedBlockNumberFinish])
			record.BlockNumber = &b
			record.Item = itemType
		default:
			logger.Panicf("unsupported item type: %d", item)
		}
		records = append(records, record)
	}

	return records, nil
}
