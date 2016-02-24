// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"time"
)

// a decoded transaction type - for JSON conversion
type Decoded struct {
	TxId        Link        `json:"txid"`
	Asset       *AssetIndex `json:"asset"`
	Exists      bool        `json:"exists"`
	State       State       `json:"state"`
	Type        string      `json:"type"`
	Transaction interface{} `json:"transaction"`
	Timestamp   *time.Time  `json:"timestamp"`
}

// decode transaction from list of ids
func Decode(txIds []Link) []Decoded {

	results := make([]Decoded, len(txIds))

	for i, txId := range txIds {

		state, data, found := txId.Read()
		results[i].TxId = txId
		results[i].State = state
		results[i].Exists = found
		results[i].Transaction = []byte(nil)
		results[i].Timestamp = nil

		if !found {
			continue // non-existant
		}

		record, err := data.Unpack()
		if nil != err {
			continue // ignore failed
		}

		switch record.(type) {
		case *AssetData:
			results[i].Type = "AssetData"
			a := record.(*AssetData).AssetIndex()
			results[i].Asset = &a
		case *BitmarkIssue:
			results[i].Type = "BitmarkIssue"
		case *BitmarkTransfer:
			results[i].Type = "BitmarkTransfer"
		default:
			results[i].Type = "?"
		}
		results[i].Transaction = record
	}

	return results
}
