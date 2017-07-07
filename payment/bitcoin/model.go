// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import "encoding/json"

const (
	bitcoin_OP_RETURN_HEX_CODE      = "6a30" // op code with 48 byte parameter
	bitcoin_OP_RETURN_PREFIX_LENGTH = len(bitcoin_OP_RETURN_HEX_CODE)
	bitcoin_OP_RETURN_PAY_ID_OFFSET = bitcoin_OP_RETURN_PREFIX_LENGTH
	bitcoin_OP_RETURN_RECORD_LENGTH = bitcoin_OP_RETURN_PREFIX_LENGTH + 2*48
)

type scriptPubKey struct {
	Hex       string   `json:"hex"`
	Addresses []string `json:"addresses"`
}

type vout struct {
	Value        json.RawMessage `json:"value"`
	ScriptPubKey scriptPubKey    `json:"scriptPubKey"`
}

type Transaction struct {
	TxID          string `json:"txid"`
	Confirmations uint64 `json:"confirmations"`
	Vout          []vout `json:"vout"`
}

type blockHeader struct {
	Hash              string `json:"hash"`
	Confirmations     uint64 `json:"confirmations"`
	Height            uint64 `json:"height"`
	Time              int64  `json:"time"`
	PreviousBlockHash string `json:"previousblockhash"`
	NextBlockHash     string `json:"nextblockhash"`
}

type block struct {
	Hash              string        `json:"hash"`
	Confirmations     uint64        `json:"confirmations"`
	Height            uint64        `json:"height"`
	Tx                []Transaction `json:"tx"`
	Time              int64         `json:"time"`
	PreviousBlockHash string        `json:"previousblockhash"`
	NextBlockHash     string        `json:"nextblockhash"`
}

type chainInfo struct {
	Version uint64 `json:"version"`
	Blocks  uint64 `json:"blocks"`
	Hash    string `json:"bestblockhash"`
}
