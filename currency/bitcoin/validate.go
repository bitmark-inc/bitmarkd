// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"bytes"
	"crypto/sha256"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// Version - to hold the type of the address
type Version byte

// AddressBytes - to hold the fixed-length address bytes
type AddressBytes [20]byte

// from: https://en.bitcoin.it/wiki/List_of_address_prefixes
const (
	Livenet       Version = 0
	LivenetScript Version = 5
	Testnet       Version = 111
	TestnetScript Version = 196
	vNull         Version = 0xff
)

// ValidateAddress - check the address and return its version
func ValidateAddress(address string) (Version, AddressBytes, error) {

	addr := util.FromBase58(address)
	addressBytes := AddressBytes{}

	if 25 != len(addr) {
		return vNull, addressBytes, fault.ErrInvalidBitcoinAddress
	}

	h := sha256.New()
	h.Write(addr[:21])
	d := h.Sum([]byte{})
	h = sha256.New()
	h.Write(d)
	d = h.Sum([]byte{})

	if !bytes.Equal(d[0:4], addr[21:]) {
		return vNull, addressBytes, fault.ErrInvalidBitcoinAddress
	}

	switch Version(addr[0]) {
	case Livenet, LivenetScript, Testnet, TestnetScript:
	default:
		return vNull, addressBytes, fault.ErrInvalidBitcoinAddress
	}

	copy(addressBytes[:], addr[1:21])

	return Version(addr[0]), addressBytes, nil
}

// IsTestnet - detect if version is a testnet value
func IsTestnet(version Version) bool {
	switch version {
	case Testnet, TestnetScript:
		return true
	default:
		return false
	}
}
