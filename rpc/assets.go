// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/asset"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Assets - type for the RPC
type Assets struct {
	log     *logger.L
	limiter *rate.Limiter
}

const (
	maximumAssets = 100
)

// AssetStatus - arguments for RPC request
type AssetStatus struct {
	AssetId   *transactionrecord.AssetIdentifier `json:"id"`
	Duplicate bool                               `json:"duplicate"`
}

// AssetsRegisterReply - results from RPC request
type AssetsRegisterReply struct {
	Assets []AssetStatus `json:"assets"`
}

// internal function to register some assets
func assetRegister(assets []*transactionrecord.AssetData) ([]AssetStatus, []byte, error) {

	assetStatus := make([]AssetStatus, len(assets))

	// pack each transaction
	packed := []byte{}
	for i, argument := range assets {

		assetId, packedAsset, err := asset.Cache(argument, storage.Pool.Assets)
		if nil != err {
			return nil, nil, err
		}

		assetStatus[i].AssetId = assetId
		if nil == packedAsset {
			assetStatus[i].Duplicate = true
		} else {
			packed = append(packed, packedAsset...)
		}
	}

	return assetStatus, packed, nil
}

// ---

// AssetGetArguments - arguments for RPC request
type AssetGetArguments struct {
	Fingerprints []string `json:"fingerprints"`
}

// AssetGetReply - results from get RPC request
type AssetGetReply struct {
	Assets []AssetRecord `json:"assets"`
}

// AssetRecord - structure of asset records in the response
type AssetRecord struct {
	Record    string      `json:"record"`
	Confirmed bool        `json:"confirmed"`
	AssetId   interface{} `json:"id,omitempty"`
	Data      interface{} `json:"data"`
}

// Get - RPC to fetch asset data
func (assets *Assets) Get(arguments *AssetGetArguments, reply *AssetGetReply) error {

	log := assets.log
	count := len(arguments.Fingerprints)

	if err := rateLimitN(assets.limiter, count, maximumAssets); nil != err {
		return err
	}

	if !mode.Is(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	log.Infof("Assets.Get: %+v", arguments)

	a := make([]AssetRecord, count)
loop:
	for i, fingerprint := range arguments.Fingerprints {

		assetId := transactionrecord.NewAssetIdentifier([]byte(fingerprint))

		confirmed := true
		_, packedAsset := storage.Pool.Assets.GetNB(assetId[:])
		if nil == packedAsset {

			confirmed = false
			packedAsset = asset.Get(assetId)
			if nil == packedAsset {
				continue loop
			}
		}

		assetTx, _, err := transactionrecord.Packed(packedAsset).Unpack(mode.IsTesting())
		if nil != err {
			continue loop
		}

		record, _ := transactionrecord.RecordName(assetTx)
		a[i] = AssetRecord{
			Record:    record,
			Confirmed: confirmed,
			AssetId:   assetId,
			Data:      assetTx,
		}
	}

	reply.Assets = a

	return nil
}

// // Asset identifier
// // -----------

// type AssetIdentifiersArguments struct {
// 	Ids []transaction.AssetId `json:"ids"`
// }

// type AssetIdentifiersReply struct {
// 	Assets []transaction.Decoded `json:"assets"`
// }

// func (assets *Assets) Identifier(arguments *AssetIdentifieresArguments, reply *AssetIdentifieresReply) error {

// 	// restrict arguments size to reasonable value
// 	size := len(arguments.Ids)
// 	if size > MaximumGetSize {
// 		size = MaximumGetSize
// 	}

// 	txIds := make([]transaction.Link, size)
// 	for i, assetId := range arguments.Ids[:size] {
// 		_, txId, found := assetId.Read()
// 		if found {
// 			txIds[i] = txId
// 		}
// 	}

// 	reply.Assets = transaction.Decode(txIds)

// 	return nil
// }
