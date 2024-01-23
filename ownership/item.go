// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ownership

import (
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// OwnedItem - the flag byte
type OwnedItem byte

// type codes for flag byte
const (
	OwnedAsset OwnedItem = iota
	OwnedBlock OwnedItem = iota
	OwnedShare OwnedItem = iota
)

// internal conversion
func toString(item OwnedItem) ([]byte, error) {
	switch item {
	case OwnedAsset:
		return []byte("Asset"), nil
	case OwnedBlock:
		return []byte("Block"), nil
	case OwnedShare:
		return []byte("Share"), nil
	default:
		return []byte{}, fault.InvalidItem
	}
}

// String - convert a owned item to its string symbol
func (item OwnedItem) String() string {
	s, err := toString(item)
	if err != nil {
		logger.Panicf("invalid item enumeration: %d", item)
	}
	return string(s)
}

// MarshalText - convert item to text
func (item OwnedItem) MarshalText() ([]byte, error) {
	s, err := toString(item)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// UnmarshalText - convert test to Item
func (item *OwnedItem) UnmarshalText(s []byte) error {
	switch strings.ToLower(string(s)) {
	case "asset":
		*item = OwnedAsset
	case "block":
		*item = OwnedBlock
	case "share":
		*item = OwnedShare
	default:
		return fault.NotOwnedItem
	}
	return nil
}
