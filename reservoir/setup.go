// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"sync"
)

// globals
type globalDataType struct {
	sync.RWMutex
	log   *logger.L
	size  int // peak size of cache
	cache map[merkle.Digest]transactionrecord.Packed
}

// gobal storage
var globalData globalDataType

// create the cache
func Initialise() error {

	globalData.Lock()
	defer globalData.Unlock()

	globalData.log = logger.New("reservoir")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.size = 0
	globalData.cache = make(map[merkle.Digest]transactionrecord.Packed, 10000)

	return nil
}

// stop all
func Finalise() {
}

// fetch a series of records
func Fetch(count int) ([]merkle.Digest, []transactionrecord.Packed, int, error) {
	if count <= 0 {
		return nil, nil, 0, fault.ErrInvalidCount
	}

	txIds := make([]merkle.Digest, 0, count)
	txData := make([]transactionrecord.Packed, 0, count)

	n := 0
	totalBytes := 0
	globalData.RLock()
	for txId, data := range globalData.cache {
		txIds = append(txIds, txId)
		txData = append(txData, data)
		totalBytes += len(data)
		n += 1
		if n >= count {
			break
		}
	}
	globalData.RUnlock()
	return txIds, txData, totalBytes, nil
}

// check if record exists
func Has(txId merkle.Digest) bool {
	globalData.RLock()
	_, ok := globalData.cache[txId]
	globalData.RUnlock()
	return ok
}

// the number of cached items
func Count() int {
	globalData.Lock()
	n := len(globalData.cache)
	globalData.Unlock()
	return n
}

// store a record
func Store(txId merkle.Digest, data transactionrecord.Packed) {
	globalData.Lock()

	globalData.cache[txId] = data

	if l := len(globalData.cache); l > globalData.size {
		globalData.size = l
		globalData.log.Infof("increased peak size to: %d", l)
	}

	globalData.Unlock()
}

// remove a record
func Delete(txId merkle.Digest) {
	globalData.Lock()
	delete(globalData.cache, txId)
	globalData.Unlock()
}
