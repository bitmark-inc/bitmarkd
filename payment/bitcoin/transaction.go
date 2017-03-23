// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/fault"
)

type bitcoinScriptPubKey struct {
	Hex       string   `json:"hex"`
	Addresses []string `json:"addresses"`
}

type bitcoinVout struct {
	Value        json.RawMessage     `json:"value"`
	ScriptPubKey bitcoinScriptPubKey `json:"scriptPubKey"`
}

type bitcoinTransaction struct {
	TxId          string        `json:"txid"`
	Confirmations uint64        `json:"confirmations"`
	Vout          []bitcoinVout `json:"vout"`
}

// fetch transaction and decode
func bitcoinGetRawTransaction(hash string, reply *bitcoinTransaction) error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	arguments := []interface{}{
		hash,
		1,
	}
	return bitcoinCall("getrawtransaction", arguments, reply)
}
