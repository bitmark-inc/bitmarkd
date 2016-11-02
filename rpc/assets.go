// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
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
	Duplicate  bool                          `json:"duplicate"`
}

type AssetsRegisterReply struct {
	Assets []AssetStatus `json:"assets"`
}

// internal function to register some assets
func assetRegister(assets []*transactionrecord.AssetData) ([]AssetStatus, []byte, error) {

	assetStatus := make([]AssetStatus, len(assets))

	// pack each transaction
	packed := []byte{}
	for i, argument := range assets {

		index, packedAsset, err := asset.Cache(argument)
		if nil != err {
			return nil, nil, err
		}

		assetStatus[i].AssetIndex = index
		if nil == packedAsset {
			assetStatus[i].Duplicate = true
		} else {
			packed = append(packed, packedAsset...)
		}
	}

	return assetStatus, packed, nil
}

// Asset get
// ---------

type AssetGetArguments struct {
	Fingerprints []string `json:"fingerprints"`
}

type AssetGetReply struct {
	Assets []AssetRecord `json:"assets"`
}

type AssetRecord struct {
	Record     string      `json:"record"`
	Confirmed  bool        `json:"confirmed"`
	AssetIndex interface{} `json:"index,omitempty"`
	Data       interface{} `json:"data"`
}

func (assets *Assets) Get(arguments *AssetGetArguments, reply *AssetGetReply) error {

	log := assets.log
	count := len(arguments.Fingerprints)

	if count > maximumAssets {
		return fault.ErrTooManyItemsToProcess
	} else if 0 == count {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	log.Infof("Assets.Get: %v", arguments)

	a := make([]AssetRecord, count)
loop:
	for i, fingerprint := range arguments.Fingerprints {

		assetIndex := transactionrecord.NewAssetIndex([]byte(fingerprint))

		confirmed := true
		packedAsset := storage.Pool.Assets.Get(assetIndex[:])
		if nil == packedAsset {

			confirmed = false
			packedAsset = asset.Get(assetIndex)
			if nil == packedAsset {
				continue loop
			}
		}

		assetTx, _, err := transactionrecord.Packed(packedAsset).Unpack()
		if nil != err {
			continue loop
		}

		record, _ := transactionrecord.RecordName(assetTx)
		a[i] = AssetRecord{
			Record:     record,
			Confirmed:  confirmed,
			AssetIndex: assetIndex,
			Data:       assetTx,
		}
	}

	reply.Assets = a

	return nil
}

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
