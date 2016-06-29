// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package asset

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
)

// the cached data
type cacheData struct {
	packed transactionrecord.Packed // data
	flag   bool                     // used to detect expired items
}

// expiry background
type expiryData struct {
	log   *logger.L
	queue chan transactionrecord.AssetIndex
}

// globals
type globalDataType struct {
	sync.RWMutex
	log        *logger.L
	expiry     expiryData
	background *background.T
	cache      map[transactionrecord.AssetIndex]*cacheData
}

// gobal storage
var globalData globalDataType

// initialise the asset cache
func Initialise() error {
	globalData.log = logger.New("asset")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	// for expiry requests, only a small queue should be sufficient
	globalData.expiry.log = logger.New("asset-expiry")
	if nil == globalData.expiry.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.expiry.queue = make(chan transactionrecord.AssetIndex, 10)

	globalData.cache = make(map[transactionrecord.AssetIndex]*cacheData)

	// list of background processes to start
	var processes = background.Processes{
		&globalData.expiry,
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// stop all background handlers
func Finalise() {

	// stop background
	globalData.background.Stop()
}

// cache an incoming asset
func Cache(asset *transactionrecord.AssetData) (*transactionrecord.AssetIndex, transactionrecord.Packed, error) {
	packedAsset, err := asset.Pack(asset.Registrant)
	if nil != err {
		return nil, nil, err
	}
	assetIndex := asset.AssetIndex()

	//***** FIX THIS: is this wanted or not
	// txId := packedAsset.MakeLink()
	// txIdBytes := txId[:]

	// already confirmed or verified
	if storage.Pool.Assets.Has(assetIndex[:]) {
		return &assetIndex, nil, nil
	}

	//***** FIX THIS: is this wanted or not
	// if storage.Pool.Transactions.Has(txIdBytes) || storage.Pool.VerifiedTransactions.Has(txIdBytes) {
	// 	return &assetIndex, nil, nil
	// }

	// create a cache entry
	d := &cacheData{
		packed: packedAsset,
		flag:   true,
	}

	// cache the record, will update partially expired item with new flag
	// causing the expiry routine to allow an extra timeout period
	globalData.Lock()
	if r, ok := globalData.cache[assetIndex]; !ok {
		globalData.cache[assetIndex] = d
	} else {
		r.flag = true     // extend timeout
		packedAsset = nil // already seen
	}
	globalData.Unlock()

	// queue for expiry
	globalData.expiry.queue <- assetIndex

	// if packedAsset is not nil then should broadcast
	return &assetIndex, packedAsset, nil
}

// check if an asset exists
func Exists(assetIndex transactionrecord.AssetIndex) bool {

	// already confirmed or verified
	if storage.Pool.Assets.Has(assetIndex[:]) {
		return true
	}

	globalData.RLock()
	_, ok := globalData.cache[assetIndex]
	globalData.RUnlock()
	return ok
}

// transfer a cached asset to permanent storage
func Store(assetIndex transactionrecord.AssetIndex) {

	// already confirmed or verified
	if storage.Pool.Assets.Has(assetIndex[:]) {
		return
	}

	// fetch the buffered data
	globalData.RLock()
	data, ok := globalData.cache[assetIndex]
	globalData.RUnlock()

	// fatal error if cache is missing
	if !ok {
		fault.Panicf("asset: Store: no cache for asset id: %v", assetIndex)
	}

	// save to permanent storage
	storage.Pool.Assets.Put(assetIndex[:], data.packed)
}
