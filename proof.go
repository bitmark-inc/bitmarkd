// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"golang.org/x/crypto/sha3"
	"time"
)

// determine the nonce as a hex string
func makeProof(payId pay.PayId, payNonce reservoir.PayNonce, difficulty string, verbose bool) string {

	nonce := uint64(12345)
	nonceBuffer := make([]byte, 8)

	start := time.Now()
	hashCount := 0

	for {
		hashCount += 1
		nonce += 113113
		binary.BigEndian.PutUint64(nonceBuffer, nonce)

		// compute hash
		h := sha3.New256()
		h.Write(payId[:])
		h.Write(payNonce[:])
		h.Write(nonceBuffer)
		var digest [32]byte
		h.Sum(digest[:0])
		if 0 == digest[0]|digest[1] {
			if verbose {
				hps := float64(hashCount) / time.Now().Sub(start).Seconds() / 1.0e6
				fmt.Printf("%f MH/s: possible nonce: %x  with digest: %x\n", hps, nonceBuffer, digest)
			}
			hexDigest := hex.EncodeToString(digest[:])
			if hexDigest <= difficulty {
				return hex.EncodeToString(nonceBuffer)
			}
		}
	}
}
