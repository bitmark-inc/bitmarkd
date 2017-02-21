// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"golang.org/x/crypto/sha3"
)

// limits
const (
	AssetIndexLength = 64
)

// the type for an asset index
// stored as little endian byte array
// represented as little endian hex text for JSON encoding
// convert a binary assetIndex to byte slice
// to get bytes value just use assetIndex[:]
type AssetIndex [AssetIndexLength]byte

// create an asset index from a byte slice
//
// SHA3-512 Hash
func NewAssetIndex(record []byte) AssetIndex {
	return AssetIndex(sha3.Sum512(record))
}

// convert a binary assetIndex to little endian hex string for use by the fmt package (for %s)
func (assetIndex AssetIndex) String() string {
	return hex.EncodeToString(assetIndex[:])
}

// convert a binary assetIndex to little endian hex string for use by the fmt package (for %#v)
func (assetIndex AssetIndex) GoString() string {
	return "<asset:" + hex.EncodeToString(assetIndex[:]) + ">"
}

// convert a little endian hex text representation to a assetIndex for use by the format package scan routines
func (assetIndex *AssetIndex) Scan(state fmt.ScanState, verb rune) error {
	token, err := state.Token(true, func(c rune) bool {
		if c >= '0' && c <= '9' {
			return true
		}
		if c >= 'A' && c <= 'F' {
			return true
		}
		if c >= 'a' && c <= 'f' {
			return true
		}
		return false
	})
	if nil != err {
		return err
	}
	if len(token) != hex.EncodedLen(AssetIndexLength) {
		return fault.ErrNotAssetIndex
	}

	byteCount, err := hex.Decode(assetIndex[:], token)
	if nil != err {
		return err
	}

	if AssetIndexLength != byteCount {
		return fault.ErrNotAssetIndex
	}
	return nil
}

// convert assetIndex to little endian hex text
func (assetIndex AssetIndex) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(assetIndex))
	buffer := make([]byte, size)
	hex.Encode(buffer, assetIndex[:])
	return buffer, nil
}

// convert little endian hex text into a assetIndex
func (assetIndex *AssetIndex) UnmarshalText(s []byte) error {
	if len(assetIndex) != hex.DecodedLen(len(s)) {
		return fault.ErrNotLink
	}
	byteCount, err := hex.Decode(assetIndex[:], s)
	if nil != err {
		return err
	}
	if AssetIndexLength != byteCount {
		return fault.ErrNotAssetIndex
	}
	return nil
}

// convert and validate little endian binary byte slice to a assetIndex
func AssetIndexFromBytes(assetIndex *AssetIndex, buffer []byte) error {
	if AssetIndexLength != len(buffer) {
		return fault.ErrNotAssetIndex
	}
	copy(assetIndex[:], buffer)
	return nil
}
