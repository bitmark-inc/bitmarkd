// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/payment/bitcoin"
	"github.com/bitmark-inc/logger"
	"golang.org/x/crypto/sha3"
	"math/big"
)

// maximum values
const (
	ReceiptLength         = 64 // hex bytes
	NonceLength           = 64 // hex bytes
	RequiredConfirmations = 3
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

// start payment tracking on an receipt
func TrackPayment(payId PayId, receipt string, confirmations uint64) {

	r, ok := get(payId)
	if !ok {
		return
	}

	hexPayId := payId.String()
	remove(payId)

	switch r.currencyName {
	case currency.Bitcoin:
		bitcoin.QueueItem(hexPayId, receipt, confirmations, r.transactions)

	default: // only fails if new module not correctly installed
		fault.Panicf("not payment handler for Currency: %s", r.currencyName.String())
	}
	//return nil
}

// instead of paying, try a proof from the client nonce
func TryProof(payId PayId, clientNonce []byte) bool {
	r, ok := get(payId)
	if !ok {
		return false // already paid/proven
	}

	if r.tracking || nil == r.difficulty { // payment tracking or proof not allowed
		return false
	}

	// save difficulty
	bigDifficulty := r.difficulty.BigInt()
	remove(payId) // remove record once done

	globalData.log.Infof("TryProof: difficulty: 0x%64x", bigDifficulty)

	// compute hash with all possible payNonces
	h := sha3.New256()
	payNonce := make([]byte, 8)
	iterator := block.NewRingReader()
	i := 0 // ***** FIX THIS: debug
	for crc, ok := iterator.Get(); ok; crc, ok = iterator.Get() {

		binary.BigEndian.PutUint64(payNonce[:], crc)
		i += 1 // ***** FIX THIS: debug
		globalData.log.Infof("TryProof: payNonce[%d]: %x", i, payNonce)

		h.Reset()
		h.Write(payId[:])
		h.Write(payNonce)
		h.Write(clientNonce)
		var digest [32]byte
		h.Sum(digest[:0])

		//globalData.log.Infof("TryProof: digest: %x", digest)

		// convert to big integer from BE byte slice
		bigDigest := new(big.Int).SetBytes(digest[:])

		globalData.log.Infof("TryProof: digest: 0x%64x", bigDigest)

		// check difficulty and verify if ok
		if bigDigest.Cmp(bigDifficulty) <= 0 {
			globalData.verifier.queue <- r.transactions
			return true
		}
	}
	return false // difficulty not reached
}
