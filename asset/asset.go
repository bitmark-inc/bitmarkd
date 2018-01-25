// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package asset

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
)

// condition of asset in the state buffer
type assetState int

// possible states
const (
	pendingState assetState = iota
	expiringState
	verifiedState
)

// the cached data
type cacheData struct {
	packed transactionrecord.Packed // data
	state  assetState               // used to detect expired/verified items
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
	globalData.log.Info("starting…")

	// for expiry requests, only a small queue should be sufficient
	globalData.expiry.log = logger.New("asset-expiry")
	globalData.expiry.queue = make(chan transactionrecord.AssetIndex, 10)

	globalData.cache = make(map[transactionrecord.AssetIndex]*cacheData)

	// list of background processes to start
	processes := background.Processes{
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

	// already confirmed
	if storage.Pool.Assets.Has(assetIndex[:]) {
		return &assetIndex, nil, nil
	}

	// create a cache entry
	d := &cacheData{
		packed: packedAsset,
		state:  pendingState,
	}

	// flag to indicate asset data would be changed
	dataWouldChange := false

	// cache the record, will update partially expired item with new flag
	// causing the expiry routine to allow an extra timeout period
	globalData.Lock()
	if r, ok := globalData.cache[assetIndex]; !ok {
		globalData.cache[assetIndex] = d
	} else {
		transaction, _, err := transactionrecord.Packed(r.packed).Unpack(mode.IsTesting())
		logger.PanicIfError("asset: bad packed record", err)

		switch tx := transaction.(type) {
		case *transactionrecord.AssetData:
			if tx.Name == asset.Name &&
				tx.Fingerprint == asset.Fingerprint &&
				tx.Metadata == asset.Metadata &&
				tx.Registrant.String() == asset.Registrant.String() {

				r.state = pendingState // extend timeout
				packedAsset = nil      // already seen
			} else {
				dataWouldChange = true
			}
		default:
			logger.Panicf("asset: non asset record in cache: %v", tx)
		}
	}
	globalData.Unlock()

	// report invalid asset changes
	if dataWouldChange {
		return nil, nil, fault.ErrAssetsAlreadyRegistered
	}

	// queue for expiry
	globalData.expiry.queue <- assetIndex

	// if packedAsset is not nil then should broadcast
	return &assetIndex, packedAsset, nil
}

// check if an asset exists
func Exists(assetIndex transactionrecord.AssetIndex) bool {

	// already confirmed
	if storage.Pool.Assets.Has(assetIndex[:]) {
		return true
	}

	globalData.RLock()
	_, ok := globalData.cache[assetIndex]
	globalData.RUnlock()
	return ok
}

// get packed asset data from cache (nil if not present)
func Get(assetIndex transactionrecord.AssetIndex) transactionrecord.Packed {

	globalData.RLock()
	item := globalData.cache[assetIndex]
	globalData.RUnlock()
	if nil == item {
		return nil
	}
	return item.packed
}

// remove an asset from the cache
func Delete(assetIndex transactionrecord.AssetIndex) {

	globalData.Lock()
	delete(globalData.cache, assetIndex)
	globalData.Unlock()
}

// mark a cached asset being verified
func SetVerified(assetIndex transactionrecord.AssetIndex) {

	// already confirmed
	if storage.Pool.Assets.Has(assetIndex[:]) {
		return
	}

	// fetch the buffered data
	globalData.RLock()
	data, ok := globalData.cache[assetIndex]
	if ok {
		// flag as verified
		data.state = verifiedState
	}
	globalData.RUnlock()

	// fatal error if cache is missing
	if !ok {
		logger.Panicf("asset: Store: no cache for asset id: %v", assetIndex)
	}
}
