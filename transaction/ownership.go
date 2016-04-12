// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"bytes"
	"encoding/binary"
)

// type to represent an ownership record
type Ownership struct {
	N         uint64 `json:"n,string"`
	TxId      Link   `json:"txid"`
	IssueTxId Link   `json:"issue"`
	AssetTxId Link   `json:"asset"`
}

// fetch a set of ownership records
func FetchOwnership(owner *Address, start uint64, count int) ([]Ownership, error) {
	if count < 1 {
		count = 1
	} else if count > 100 {
		count = 100
	}

	n := make([]byte, 8)
	binary.BigEndian.PutUint64(n, start)

	cursor := transactionPool.ownershipPool.NewFetchCursor().Seek(append(owner.PublicKeyBytes(), n...))

	items, err := cursor.Fetch(count)
	if nil != err {
		return nil, err
	}

	ownership := make([]Ownership, 0, len(items))

	currentOwner := owner.PublicKeyBytes()

loop:
	for _, item := range items {

		nStart := len(item.Key) - 8

		// if over the end of current owners data
		if !bytes.Equal(currentOwner, item.Key[:nStart]) {
			break loop
		}

		n := binary.BigEndian.Uint64(item.Key[nStart:])

		var txid Link
		var issue Link
		var asset Link
		LinkFromBytes(&txid, item.Value[:LinkSize])
		LinkFromBytes(&issue, item.Value[LinkSize:2*LinkSize])
		LinkFromBytes(&asset, item.Value[2*LinkSize:])

		r := Ownership{
			N:         n,
			TxId:      txid,
			IssueTxId: issue,
			AssetTxId: asset,
		}
		ownership = append(ownership, r)
	}

	return ownership, nil
}
