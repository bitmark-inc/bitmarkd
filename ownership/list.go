// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ownership

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// type to represent an ownership record
type Ownership struct {
	N           uint64                       `json:"n,string"`
	TxId        merkle.Digest                `json:"txId"`
	IssueTxId   merkle.Digest                `json:"issue"`
	Item        OwnedItem                    `json:"item"`
	AssetIndex  transactionrecord.AssetIndex `json:"index"`
	BlockNumber uint64                       `json:"blockNumber"`
}

// fetch a list of bitmarks for an owner
func ListBitmarksFor(owner *account.Account, start uint64, count int) ([]Ownership, error) {

	startBytes := make([]byte, uint64ByteSize)
	binary.BigEndian.PutUint64(startBytes, start)
	prefix := append(owner.Bytes(), startBytes...)

	cursor := storage.Pool.Ownership.NewFetchCursor().Seek(prefix)

	items, err := cursor.Fetch(count)
	if nil != err {
		return nil, err
	}

	records := make([]Ownership, len(items))

	for i, item := range items {
		n := len(item.Key)
		records[i].N = binary.BigEndian.Uint64(item.Key[n-uint64ByteSize:])
		merkle.DigestFromBytes(&records[i].TxId, item.Value[TxIdStart:TxIdFinish])
		merkle.DigestFromBytes(&records[i].IssueTxId, item.Value[IssueTxIdStart:IssueTxIdFinish])

		switch itemType := OwnedItem(item.Value[FlagByteStart]); itemType {
		case OwnedAsset:
			transactionrecord.AssetIndexFromBytes(&records[i].AssetIndex, item.Value[AssetIndexStart:AssetIndexFinish])
			records[i].Item = itemType
		case OwnedBlock:
			records[i].BlockNumber = binary.BigEndian.Uint64(item.Value[OwnedBlockNumberStart:OwnedBlockNumberFinish])
			records[i].Item = itemType
		default:
			logger.Panicf("unsupported item type: %d", item)
		}
	}

	return records, nil
}
