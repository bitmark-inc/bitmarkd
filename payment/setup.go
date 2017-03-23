// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	//"encoding/binary"
	//"github.com/bitmark-inc/bitmarkd/blockring"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	//"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/payment/bitcoin"
	"github.com/bitmark-inc/logger"
	//"golang.org/x/crypto/sha3"
	//"math/big"
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

// globals
type globalDataType struct {
	log *logger.L
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

	// initialise all currency handlers
	for c := currency.First; c <= currency.Last; c += 1 {
		switch c {
		case currency.Bitcoin:
			err := bitcoin.Initialise(configuration.Bitcoin)
			if nil != err {
				return err
			}
		default: // only fails if new module not correctly installed
			fault.Panicf("missing payment initialiser for Currency: %s", c.String())
		}
	}

	return nil
}

// stop all payment handlers
func Finalise() {

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// finalise all currency handlers
	for c := currency.First; c <= currency.Last; c += 1 {
		switch c {
		case currency.Bitcoin:
			bitcoin.Finalise()
		default: // only fails if new module not correctly installed
			fault.Panicf("missing payment finaliser for Currency: %s", c.String())
		}
	}

	globalData.log.Info("finished")
	globalData.log.Flush()
}

// // store an incoming record for payment
// func Store(payments []*transactionrecord.Payment, payId pay.PayId, count int, canProof bool) (PayNonce, *difficulty.Difficulty, error) {

// 	payNonce := NewPayNonce()

// 	if nil == payments || 0 == len(payments) {
// 		payments = nil // for consistency
// 	} else {
// 		// ensure all payments have the same currency
// 		first := payments[0].Currency
// 		for _, c := range payments[1:] {
// 			if first != c.Currency {
// 				return payNonce, nil, fault.ErrInvalidMixedCurrencyPayment
// 			}
// 		}
// 	}

// 	u := &unverified{
// 		payments:   payments,
// 		difficulty: nil,
// 		done:       false,
// 	}

// 	// only create difficulty if proof is allowed
// 	if canProof {
// 		d := ScaledDifficulty(count)
// 		u.difficulty = d
// 	}

// 	// cache the record
// 	newItem := put(payId, u)

// 	// add an expire
// 	if newItem {
// 		globalData.expiry.queue <- payId
// 	}

// 	return payNonce, u.difficulty, nil

// }

// // ***** FIX THIS: REMOVE THIS:---> version 4.x
// // start payment tracking on an receipt
// func TrackPayment(payId pay.PayId, receipt string, confirmations uint64) TrackingStatus {

// 	r, done, ok := get(payId)
// 	if !ok {
// 		return TrackingNotFound
// 	}
// 	if done {
// 		return TrackingProcessed
// 	}

// 	// if no payment required
// 	if nil == r.payments {
// 		return TrackingInvalid
// 	}

// 	status := TrackingInvalid

// 	c := r.payments[0].Currency
// 	switch c {
// 	case currency.Bitcoin:
// 		status = TrackingAccepted

// 	default: // only fails if new module not correctly installed
// 		fault.Panicf("no payment handler for Currency: %s", c.String())
// 	}
// 	return status
// }

// // instead of paying, try a proof from the client nonce
// func TryProof(payId pay.PayId, clientNonce []byte) TrackingStatus {

// 	r, done, ok := get(payId)
// 	if !ok {
// 		return TrackingNotFound
// 	}
// 	if done {
// 		return TrackingProcessed
// 	}
// 	if nil == r.difficulty { // only payment tracking; proof not allowed
// 		return TrackingInvalid
// 	}

// 	// convert difficulty
// 	bigDifficulty := r.difficulty.BigInt()

// 	globalData.log.Infof("TryProof: difficulty: 0x%064x", bigDifficulty)

// 	// compute hash with all possible payNonces
// 	h := sha3.New256()
// 	payNonce := make([]byte, 8)
// 	iterator := blockring.NewRingReader()
// 	i := 0 // ***** FIX THIS: debug
// 	for crc, ok := iterator.Get(); ok; crc, ok = iterator.Get() {

// 		binary.BigEndian.PutUint64(payNonce[:], crc)
// 		i += 1 // ***** FIX THIS: debug
// 		globalData.log.Debugf("TryProof: payNonce[%d]: %x", i, payNonce)

// 		h.Reset()
// 		h.Write(payId[:])
// 		h.Write(payNonce)
// 		h.Write(clientNonce)
// 		var digest [32]byte
// 		h.Sum(digest[:0])

// 		//globalData.log.Debugf("TryProof: digest: %x", digest)

// 		// convert to big integer from BE byte slice
// 		bigDigest := new(big.Int).SetBytes(digest[:])

// 		globalData.log.Debugf("TryProof: digest: 0x%064x", bigDigest)

// 		// check difficulty and verify if ok
// 		if bigDigest.Cmp(bigDifficulty) <= 0 {
// 			globalData.log.Debugf("TryProof: success: pay id: %s", payId)
// 			globalData.verifier.queue <- payId
// 			return TrackingAccepted
// 		}
// 	}
// 	return TrackingInvalid
// }
