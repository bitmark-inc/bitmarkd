// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// limits
const (
	AssetIdentifierLength = 64
)

// AssetIdentifier - the type for an asset identifier
// stored as little endian byte array
// represented as little endian hex text for JSON encoding
// convert a binary assetId to byte slice
// to get bytes value just use assetId[:]
type AssetIdentifier [AssetIdentifierLength]byte

// NewAssetIdentifier - create an asset id from a byte slice
//
// SHA3-512 Hash
func NewAssetIdentifier(record []byte) AssetIdentifier {
	return AssetIdentifier(sha3.Sum512(record))
}

// String - convert a binary assetId to little endian hex string for use by the fmt package (for %s)
func (assetId AssetIdentifier) String() string {
	return hex.EncodeToString(assetId[:])
}

// GoString - convert a binary assetId to little endian hex string for use by the fmt package (for %#v)
func (assetId AssetIdentifier) GoString() string {
	return "<asset:" + hex.EncodeToString(assetId[:]) + ">"
}

// Scan - convert a little endian hex text representation to a assetId for use by the format package scan routines
func (assetId *AssetIdentifier) Scan(state fmt.ScanState, verb rune) error {
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
	if len(token) != hex.EncodedLen(AssetIdentifierLength) {
		return fault.NotAssetId
	}

	byteCount, err := hex.Decode(assetId[:], token)
	if nil != err {
		return err
	}

	if AssetIdentifierLength != byteCount {
		return fault.NotAssetId
	}
	return nil
}

// MarshalText - convert assetId to little endian hex text
func (assetId AssetIdentifier) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(assetId))
	buffer := make([]byte, size)
	hex.Encode(buffer, assetId[:])
	return buffer, nil
}

// UnmarshalText - convert little endian hex text into a assetId
func (assetId *AssetIdentifier) UnmarshalText(s []byte) error {
	if len(assetId) != hex.DecodedLen(len(s)) {
		return fault.NotLink
	}
	byteCount, err := hex.Decode(assetId[:], s)
	if nil != err {
		return err
	}
	if AssetIdentifierLength != byteCount {
		return fault.NotAssetId
	}
	return nil
}

// AssetIdentifierFromBytes - convert and validate little endian binary byte slice to a assetId
func AssetIdentifierFromBytes(assetId *AssetIdentifier, buffer []byte) error {
	if AssetIdentifierLength != len(buffer) {
		return fault.NotAssetId
	}
	copy(assetId[:], buffer)
	return nil
}
