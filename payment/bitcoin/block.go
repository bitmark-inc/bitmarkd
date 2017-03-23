// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

type bitcoinBlock struct {
	Hash              string   `json:"hash"`
	Confirmations     uint64   `json:"confirmations"`
	Size              uint64   `json:"size"`
	Height            uint64   `json:"height"`
	Version           uint64   `json:"version"`
	MerkleRoot        string   `json:"merkleroot"`
	Tx                []string `json:"tx"`
	Time              uint64   `json:"time"`
	MedianTime        uint64   `json:"mediantime"`
	Nonce             uint64   `json:"nonce"`
	Bits              string   `json:"bits"`
	Difficulty        float64  `json:"difficulty"`
	ChainWork         string   `json:"chainwork"`
	PreviousBlockHash string   `json:"previousblockhash"`
	NextBlockHash     string   `json:"nextblockhash"`
}

// fetch block hash
func bitcoinGetBlockHash(blockNumber uint64, hash *string) error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	arguments := []interface{}{
		blockNumber,
	}
	return bitcoinCall("getblockhash", arguments, hash)
}

// fetch block and decode
func bitcoinGetBlock(hash string, reply *bitcoinBlock) error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	arguments := []interface{}{
		hash,
	}
	return bitcoinCall("getblock", arguments, reply)
}
