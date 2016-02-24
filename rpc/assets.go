// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

// Assets type
// ------=----

type Assets struct {
	log   *logger.L
	asset *Asset
}

// Assets registration
// -------------------

func (assets *Assets) Register(arguments *[]transaction.AssetData, reply *[]AssetRegisterReply) error {

	asset := assets.asset

	result := make([]AssetRegisterReply, len(*arguments))
	for i, argument := range *arguments {
		if err := asset.Register(&argument, &result[i]); err != nil {
			result[i].Err = err.Error()
		}
	}

	*reply = result
	return nil
}

// Asset get
// ---------

type AssetGetArguments struct {
	Fingerprints []string `json:"fingerprints"`
}

type AssetGetReply struct {
	Assets []transaction.Decoded `json:"assets"`
}

func (assets *Assets) Get(arguments *AssetGetArguments, reply *AssetGetReply) error {

	// restrict arguments size to reasonable value
	size := len(arguments.Fingerprints)
	if size > MaximumGetSize {
		size = MaximumGetSize
	}

	txIds := make([]transaction.Link, size)
	for i, fingerprint := range arguments.Fingerprints[:size] {
		assetIndex := transaction.NewAssetIndex([]byte(fingerprint))
		_, txid, found := assetIndex.Read()
		if found {
			txIds[i] = txid
		}
	}

	reply.Assets = transaction.Decode(txIds)

	return nil
}

// Asset index
// -----------

type AssetIndexesArguments struct {
	Indexes []transaction.AssetIndex `json:"indexes"`
}

type AssetIndexesReply struct {
	Assets []transaction.Decoded `json:"assets"`
}

func (assets *Assets) Index(arguments *AssetIndexesArguments, reply *AssetIndexesReply) error {

	// restrict arguments size to reasonable value
	size := len(arguments.Indexes)
	if size > MaximumGetSize {
		size = MaximumGetSize
	}

	txIds := make([]transaction.Link, size)
	for i, assetIndex := range arguments.Indexes[:size] {
		_, txid, found := assetIndex.Read()
		if found {
			txIds[i] = txid
		}
	}

	reply.Assets = transaction.Decode(txIds)

	return nil
}
