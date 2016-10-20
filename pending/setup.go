// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pending

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/logger"
	"sync"
	"time"
)

// expiry background
type expiryData struct {
	log *logger.L
}

// number of table shards must be a power of 2
// and mask is the corresponding bit mask
// only the first byte of the key is used
const (
	shards = 16         // maximum value: 256
	mask   = shards - 1 // bit mask
)

// data items
type dataItem struct {
	txId      merkle.Digest
	timestamp time.Time
}

// lockable map
type lockable struct {
	sync.RWMutex
	table map[merkle.Digest]dataItem
}

// globals
type globalDataType struct {
	sync.RWMutex
	log   *logger.L
	cache [shards]lockable

	expiry     expiryData
	background *background.T
}

// gobal storage
var globalData globalDataType

// create the cache
func Initialise() error {

	globalData.Lock()
	defer globalData.Unlock()

	globalData.log = logger.New("pending")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.expiry.log = logger.New("pending")
	if nil == globalData.expiry.log {
		return fault.ErrInvalidLoggerChannel
	}

	for i := 0; i < shards; i += 1 {
		globalData.cache[i] = lockable{
			table: make(map[merkle.Digest]dataItem, 10000),
		}
	}

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	var processes = background.Processes{
		&globalData.expiry,
	}

	globalData.background = background.Start(processes, &globalData)

	return nil
}

// stop all
func Finalise() {
	// stop background
	globalData.background.Stop()
}

// store a record
func Add(link merkle.Digest, txId merkle.Digest) error {

	// result
	err := error(nil)

	// select the table
	n := link[0] & mask

	// need a full lock as this is a write
	// (no defer as overhead is too high for such a short routine)
	globalData.cache[n].Lock()
	record, ok := globalData.cache[n].table[link]
	if ok {
		// already exists, just update timestamp
		record.timestamp = time.Now()
		if txId != record.txId {
			err = fault.ErrDoubleTransferAttempt
		}
	} else {
		// create a new entry
		globalData.cache[n].table[link] = dataItem{
			txId:      txId,
			timestamp: time.Now(),
		}
	}
	globalData.cache[n].Unlock()

	return err
}

// remove a record
func Remove(link merkle.Digest) (merkle.Digest, bool) {

	// select the table
	n := link[0] & mask

	// need a full lock as this is a write
	// (no defer as overhead is too high for such a short routine)
	globalData.cache[n].Lock()
	record, ok := globalData.cache[n].table[link]
	delete(globalData.cache[n].table, link)
	globalData.cache[n].Unlock()
	return record.txId, ok
}
