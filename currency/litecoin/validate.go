// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin

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
	Livenet        Version = 48
	LivenetScript  Version = 5
	LivenetScript2 Version = 50
	Testnet        Version = 111
	TestnetScript  Version = 196
	TestnetScript2 Version = 58
	vNull          Version = 0xff
)

// ValidateAddress - check the address and return its version
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
	case Livenet, LivenetScript, LivenetScript2, Testnet, TestnetScript, TestnetScript2:
		// OK
	default:
		return vNull, addressBytes, fault.ErrInvalidLitecoinAddress
	}

	copy(addressBytes[:], addr[1:21])

	return Version(addr[0]), addressBytes, nil
}

// TransformAddress - convert address to/from new version prefix
func TransformAddress(address string) (string, error) {
	version, addressBytes, err := ValidateAddress(address)
	if nil != err {
		return "", err
	}
	switch version {
	case Livenet:
		return address, nil
	case LivenetScript:
		return compose(LivenetScript2, addressBytes), nil
	case LivenetScript2:
		return compose(LivenetScript, addressBytes), nil
	case Testnet:
		return address, nil
	case TestnetScript:
		return compose(TestnetScript2, addressBytes), nil
	case TestnetScript2:
		return compose(TestnetScript, addressBytes), nil
	default:
		return "", fault.ErrInvalidLitecoinAddress
	}
}

// IsTestnet - detect if version is a testnet value
func IsTestnet(version Version) bool {
	switch version {
	case Testnet, TestnetScript, TestnetScript2:
		return true
	default:
		return false
	}
}

// build address
func compose(version Version, addressBytes AddressBytes) string {

	addr := append([]byte{byte(version)}, addressBytes[:]...)

	h := sha256.New()
	h.Write(addr)
	d := h.Sum([]byte{})
	h = sha256.New()
	h.Write(d)
	d = h.Sum([]byte{})

	addr = append(addr, d[0:4]...)
	return util.ToBase58(addr)
}
