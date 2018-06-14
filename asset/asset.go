// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package asset

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
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
	queue chan transactionrecord.AssetIdentifier
}

// globals
type globalDataType struct {
	sync.RWMutex
	log        *logger.L
	expiry     expiryData
	background *background.T
	cache      map[transactionrecord.AssetIdentifier]*cacheData
}

// gobal storage
var globalData globalDataType

// initialise the asset cache
func Initialise() error {
	globalData.log = logger.New("asset")
	globalData.log.Info("starting…")

	// for expiry requests, only a small queue should be sufficient
	globalData.expiry.log = logger.New("asset-expiry")
	globalData.expiry.queue = make(chan transactionrecord.AssetIdentifier, 10)

	globalData.cache = make(map[transactionrecord.AssetIdentifier]*cacheData)

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
func Cache(asset *transactionrecord.AssetData) (*transactionrecord.AssetIdentifier, transactionrecord.Packed, error) {
	packedAsset, err := asset.Pack(asset.Registrant)
	if nil != err {
		return nil, nil, err
	}
	assetId := asset.AssetId()

	// already confirmed
	if storage.Pool.Assets.Has(assetId[:]) {
		return &assetId, nil, nil
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
	if r, ok := globalData.cache[assetId]; !ok {
		globalData.cache[assetId] = d
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
	globalData.expiry.queue <- assetId

	// if packedAsset is not nil then should broadcast
	return &assetId, packedAsset, nil
}

// check if an asset exists
func Exists(assetId transactionrecord.AssetIdentifier) bool {

	// already confirmed
	if storage.Pool.Assets.Has(assetId[:]) {
		return true
	}

	globalData.RLock()
	_, ok := globalData.cache[assetId]
	globalData.RUnlock()
	return ok
}

// get packed asset data from cache (nil if not present)
func Get(assetId transactionrecord.AssetIdentifier) transactionrecord.Packed {

	globalData.RLock()
	item := globalData.cache[assetId]
	globalData.RUnlock()
	if nil == item {
		return nil
	}
	return item.packed
}

// remove an asset from the cache
func Delete(assetId transactionrecord.AssetIdentifier) {

	globalData.Lock()
	delete(globalData.cache, assetId)
	globalData.Unlock()
}

// mark a cached asset being verified
func SetVerified(assetId transactionrecord.AssetIdentifier) {

	// already confirmed
	if storage.Pool.Assets.Has(assetId[:]) {
		return
	}

	// fetch the buffered data
	globalData.RLock()
	data, ok := globalData.cache[assetId]
	if ok {
		// flag as verified
		data.state = verifiedState
	}
	globalData.RUnlock()

	// fatal error if cache is missing
	if !ok {
		logger.Panicf("asset: Store: no cache for asset id: %v", assetId)
	}
}
