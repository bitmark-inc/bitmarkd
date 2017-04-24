// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"bytes"
	"crypto/sha256"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// to hold the type of the address
type Version byte

// from: https://en.bitcoin.it/wiki/List_of_address_prefixes
const (
	Livenet       Version = 0
	LivenetScript Version = 5
	Testnet       Version = 111
	TestnetScript Version = 196
	vNull         Version = 0xff
)

// check the address ad return its version
func ValidateAddress(address string) (Version, error) {

	addr := util.FromBase58(address)

	if 25 != len(addr) {
		return vNull, fault.ErrInvalidBitcoinAddress
	}

	h := sha256.New()
	h.Write(addr[:21])
	d := h.Sum([]byte{})
	h = sha256.New()
	h.Write(d)
	d = h.Sum([]byte{})

	if !bytes.Equal(d[0:4], addr[21:]) {
		return vNull, fault.ErrInvalidBitcoinAddress
	}
	return Version(addr[0]), nil
}
