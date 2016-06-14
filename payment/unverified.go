// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/payment/bitcoin"
	"github.com/bitmark-inc/logger"
	"golang.org/x/crypto/sha3"
	"math/big"
	"sync"
)

// a set of transactions awaiting payment
type unverified struct {
	currencyName currency.Currency      // currency identifier
	payNonce     PayNonce               // for client-side hashing
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

// // to control expiry
// type expiry struct {
// 	payId   PayId     // item to remove
// 	expires time.Time // remove the record after this time
// }
// need a simple FIFO queue

// number of table shards must be a power of 2
// and mask is the corresponding bit mask
// only the first byte of the key is used
const (
	shards = 16         // maximum value: 256
	mask   = shards - 1 // bit mask
)

// array of tables to reduce contention
var cache [shards]lockable

// configuration for each sub-module
type Configuration struct {
	Bitcoin *bitcoin.Configuration
}

// for background
type expiryData struct {
	log        *logger.L
	queue      chan PayId
	background *background.T
}

// background task
var globalData expiryData

// create the tables
func Initialise(configuration *Configuration) error {
	globalData.log = logger.New("payment")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	// create initial cache
	for i := 0; i < len(cache); i += 1 {
		cache[i].table = make(waiting)
	}

	// initialise all currency handlers
	for c := currency.First; c <= currency.Last; c += 1 {
		switch c {
		case currency.Bitcoin:
			bitcoin.Initialise(configuration.Bitcoin)
		default: // only fails if new module not correctly installed
			fault.Panicf("not payment initialiser for Currency: %s", c.String())
		}
	}

	// for expiry requests, only a small queue should be sufficient
	globalData.queue = make(chan PayId, 10)

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	var processes = background.Processes{
		&globalData,
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// stop all payment handlers
func Finalise() {

	// stop background
	globalData.background.Stop()

	// finalise all currency handlers
	for c := currency.First; c <= currency.Last; c += 1 {
		switch c {
		case currency.Bitcoin:
			bitcoin.Finalise()
		default: // only fails if new module not correctly installed
			fault.Panicf("not payment finaliser for Currency: %s", c.String())
		}
	}
}

// store an incoming record for payment
func Store(currencyName currency.Currency, transactions []byte, count int, canProof bool) (PayId, PayNonce, *difficulty.Difficulty) {
	payId := NewPayId(transactions)
	payNonce := NewPayNonce()

	t := make([]byte, len(transactions))
	copy(t, transactions) // copy to preserve underlying data

	u := &unverified{
		currencyName: currencyName,
		payNonce:     payNonce,
		difficulty:   nil,
		tracking:     false,
		transactions: t,
	}

	globalData.queue <- payId

	// only create difficulty if proof is allowed
	if canProof {
		d := ScaledDifficulty(count)
		u.difficulty = d
	}

	put(payId, u) // ***** FIX THIS: need a way to expire
	return payId, payNonce, u.difficulty

}

// start payment tracking on an id
func TrackPayment(payId PayId, txId string, confirmations uint64) {

	r, ok := get(payId)
	if !ok {
		return
		//return fault.ErrRecordNotFound
	}

	r.tracking = true // enable tracking
	hexPayId := payId.String()

	switch r.currencyName {
	case currency.Bitcoin:
		bitcoin.QueueItem(hexPayId, txId, confirmations)

	default: // only fails if new module not correctly installed
		fault.Panicf("not payment handler for Currency: %s", r.currencyName.String())
	}
	//return nil
}

// instead of paying, try a proof
func TryProof(payId PayId, nonce []byte) bool {
	r, ok := get(payId)
	if !ok {
		return false // already paid/proven
	}

	if r.tracking || nil == r.difficulty { // payment tracking or proof not allowed
		return false
	}

	// compute hash
	h := sha3.New256()
	h.Write(payId[:])
	h.Write(r.payNonce[:])
	h.Write(nonce)
	var digest [32]byte
	h.Sum(digest[:0]) // ***** FIX THIS: should this be LE, (currently assumed as BE)
	// ***** FIX THIS: reverse digest?   ^^^^^^^^^^^^^^^^^

	remove(payId) // remove record once done

	// convert to big integer from BE byte slice
	bigDigest := new(big.Int).SetBytes(digest[:])

	// check difficulty
	if bigDigest.Cmp(r.difficulty.BigInt()) > 0 {
		return false // difficult not reached
	}
	return true
}

// store the payRecord in the cache
func put(payId PayId, r *unverified) {
	// select the table
	n := payId[0] & mask

	// need a full lock as this is a write
	// (no defer as overhead is too high for such a short routine)
	cache[n].Lock()
	cache[n].table[payId] = r
	cache[n].Unlock()
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
