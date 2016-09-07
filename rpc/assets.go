// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Assets type
// ------=----

type Assets struct {
	log *logger.L
}

const (
	maximumAssets = 100
)

// Assets registration
// -------------------
type AssetStatus struct {
	AssetIndex *transactionrecord.AssetIndex `json:"index"`
	Duplicate  bool                          `json:"duplicate"` // ***** FIX THIS: is this necessary?
}

type AssetsRegisterReply struct {
	Assets []AssetStatus `json:"assets"`
}

func (assets *Assets) Register(arguments *[]transactionrecord.AssetData, reply *AssetsRegisterReply) error {

	log := assets.log
	count := len(*arguments)

	if count > maximumAssets {
		return fault.ErrTooManyItemsToProcess
	} else if 0 == count {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	log.Infof("Assets.Register: %v", arguments)

	result := AssetsRegisterReply{
		Assets: make([]AssetStatus, count),
	}

	// pack each transaction
	packed := []byte{}
	for i, argument := range *arguments {

		index, packedAsset, err := asset.Cache(&argument)
		if nil != err {
			return err
		}

		result.Assets[i].AssetIndex = index
		if nil == packedAsset {
			result.Assets[i].Duplicate = true
		} else {
			packed = append(packed, packedAsset...)
		}
	}

	// fail if no data sent
	if 0 == len(packed) {
		return fault.ErrAssetsAlreadyRegistered
	}

	// announce transaction block to other peers
	messagebus.Bus.Broadcast.Send("assets", packed)

	*reply = result
	return nil
}

// Asset get
// ---------

// type AssetGetArguments struct {
// 	Fingerprints []string `json:"fingerprints"`
// }

// type AssetGetReply struct {
// 	Assets []transaction.Decoded `json:"assets"`
// }

// func (assets *Assets) Get(arguments *AssetGetArguments, reply *AssetGetReply) error {

// 	// restrict arguments size to reasonable value
// 	size := len(arguments.Fingerprints)
// 	if size > MaximumGetSize {
// 		size = MaximumGetSize
// 	}

// 	txIds := make([]transaction.Link, size)
// 	for i, fingerprint := range arguments.Fingerprints[:size] {
// 		assetIndex := transaction.NewAssetIndex([]byte(fingerprint))
// 		_, txId, found := assetIndex.Read()
// 		if found {
// 			txIds[i] = txId
// 		}
// 	}

// 	reply.Assets = transaction.Decode(txIds)

// 	return nil
// }

// // Asset index
// // -----------

// type AssetIndexesArguments struct {
// 	Indexes []transaction.AssetIndex `json:"indexes"`
// }

// type AssetIndexesReply struct {
// 	Assets []transaction.Decoded `json:"assets"`
// }

// func (assets *Assets) Index(arguments *AssetIndexesArguments, reply *AssetIndexesReply) error {

// 	// restrict arguments size to reasonable value
// 	size := len(arguments.Indexes)
// 	if size > MaximumGetSize {
// 		size = MaximumGetSize
// 	}

// 	txIds := make([]transaction.Link, size)
// 	for i, assetIndex := range arguments.Indexes[:size] {
// 		_, txId, found := assetIndex.Read()
// 		if found {
// 			txIds[i] = txId
// 		}
// 	}

// 	reply.Assets = transaction.Decode(txIds)

// 	return nil
// }
