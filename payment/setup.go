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
)

// configuration for each sub-module
type Configuration struct {
	Bitcoin *bitcoin.Configuration
}

// expiry background
type expiryData struct {
	log   *logger.L
	queue chan PayId
}

// verifier background
type verifierData struct {
	log   *logger.L
	queue chan []byte
}

// globals
type globalDataType struct {
	log        *logger.L
	expiry     expiryData
	verifier   verifierData
	background *background.T
}

// gobal storage
var globalData globalDataType

// create the tables
func Initialise(configuration *Configuration) error {
	globalData.log = logger.New("payment")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	// for expiry requests, only a small queue should be sufficient
	globalData.expiry.log = logger.New("payment-expiry")
	if nil == globalData.expiry.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.expiry.queue = make(chan PayId, 10)

	// for verifier
	globalData.verifier.log = logger.New("payment-verifier")
	if nil == globalData.verifier.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.verifier.queue = make(chan []byte, 10)

	// initialise all currency handlers
	for c := currency.First; c <= currency.Last; c += 1 {
		switch c {
		case currency.Bitcoin:
			bitcoin.Initialise(configuration.Bitcoin, globalData.verifier.queue)
		default: // only fails if new module not correctly installed
			fault.Panicf("not payment initialiser for Currency: %s", c.String())
		}
	}

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	var processes = background.Processes{
		&globalData.expiry,
		&globalData.verifier,
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

	// only create difficulty if proof is allowed
	if canProof {
		d := ScaledDifficulty(count)
		u.difficulty = d
	}

	// cache the record
	put(payId, u)

	// add an expire
	globalData.expiry.queue <- payId

	return payId, payNonce, u.difficulty

}

// start payment tracking on an id
func TrackPayment(payId PayId, txId string, confirmations uint64) {

	r, ok := get(payId)
	if !ok {
		return
	}

	hexPayId := payId.String()
	remove(payId)

	switch r.currencyName {
	case currency.Bitcoin:
		bitcoin.QueueItem(hexPayId, txId, confirmations, r.transactions)

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

	globalData.log.Infof("TryProof: digest: %x", digest)

	remove(payId) // remove record once done

	// convert to big integer from BE byte slice
	bigDigest := new(big.Int).SetBytes(digest[:])

	// check difficulty
	if bigDigest.Cmp(r.difficulty.BigInt()) > 0 {
		return false // difficult not reached
	}
	globalData.verifier.queue <- r.transactions
	return true
}
