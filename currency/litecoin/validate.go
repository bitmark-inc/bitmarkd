// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin

import (
	"bytes"
	"crypto/sha256"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// to hold the type of the address
type Version byte

// to hold thefixed-length address bytes
type AddressBytes [20]byte

// from: https://en.bitcoin.it/wiki/List_of_address_prefixes
const (
	Livenet        Version = 48
	LivenetScript  Version = 5
	LivenetScript2 Version = 50
	Testnet        Version = 111
	TestnetScript  Version = 196
	vNull          Version = 0xff
)

// check the address and return its version
func ValidateAddress(address string) (Version, AddressBytes, error) {

	addr := util.FromBase58(address)
	addressBytes := AddressBytes{}

	if 25 != len(addr) {
		return vNull, addressBytes, fault.ErrInvalidLitecoinAddress
	}

	h := sha256.New()
	h.Write(addr[:21])
	d := h.Sum([]byte{})
	h = sha256.New()
	h.Write(d)
	d = h.Sum([]byte{})

	if !bytes.Equal(d[0:4], addr[21:]) {
		return vNull, addressBytes, fault.ErrInvalidLitecoinAddress
	}

	switch Version(addr[0]) {
	case Livenet, LivenetScript, LivenetScript2, Testnet, TestnetScript:
	default:
		return vNull, addressBytes, fault.ErrInvalidLitecoinAddress
	}

	copy(addressBytes[:], addr[1:21])

	return Version(addr[0]), addressBytes, nil
}

// detect if version is a testnet value
func IsTestnet(version Version) bool {
	switch version {
	case Testnet, TestnetScript:
		return true
	default:
		return false
	}
}
