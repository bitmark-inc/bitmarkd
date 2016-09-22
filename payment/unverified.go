// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"sync"
)

// a set of transactions awaiting payment
type unverified struct {
	currencyName currency.Currency      // currency identifier
	tracking     bool                   // payment tracking requested
	difficulty   *difficulty.Difficulty // for proof request
	transactions []byte                 // all the transactions in this payment set
}

// a set of items awaiting payment
type waiting map[PayId]*unverified

// lockable map
type lockable struct {
	sync.RWMutex
	table waiting
}

// number of table shards must be a power of 2
// and mask is the corresponding bit mask
// only the first byte of the key is used
const (
	shards = 16         // maximum value: 256
	mask   = shards - 1 // bit mask
)

// array of tables to reduce contention
var cache [shards]lockable

// create initial cache
func init() {
	for i := 0; i < len(cache); i += 1 {
		cache[i].table = make(waiting)
	}
}

// store the payRecord in the cache
//
// returns true if newly cached item
func put(payId PayId, r *unverified) bool {
	// select the table
	n := payId[0] & mask

	// need a full lock as this is a write
	// (no defer as overhead is too high for such a short routine)
	cache[n].Lock()
	_, ok := cache[n].table[payId]
	cache[n].table[payId] = r
	cache[n].Unlock()
	return !ok
}

// read the payRecord from the cache
func get(payId PayId) (*unverified, bool) {
	// select the table
	n := payId[0] & mask

	// only need a read lock
	// (no defer as overhead is too high for such a short routine)
	cache[n].RLock()
	r, ok := cache[n].table[payId]
	cache[n].RUnlock()
	return r, ok
}

// remove the payRecord from the cache
func remove(payId PayId) {
	// select the table
	n := payId[0] & mask

	// need a full lock as this is a write
	// (no defer as overhead is too high for such a short routine)
	cache[n].Lock()
	delete(cache[n].table, payId)
	cache[n].Unlock()
}
