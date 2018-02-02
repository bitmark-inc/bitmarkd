// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ownership

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// the flag byte
type OwnedItem byte

// type codes for flag byte
const (
	OwnedAsset OwnedItem = iota
	OwnedBlock OwnedItem = iota
)

// internal conversion
func toString(item OwnedItem) ([]byte, error) {
	switch item {
	case OwnedAsset:
		return []byte("Asset"), nil
	case OwnedBlock:
		return []byte("Block"), nil
	default:
		return []byte{}, fault.ErrInvalidItem
	}
}

// convert a currency to its string symbol
func (item OwnedItem) String() string {
	s, err := toString(item)
	if nil != err {
		logger.Panicf("invalid item enumeration: %d", item)
	}
	return string(s)
}

// convert item to  text
func (item OwnedItem) MarshalText() ([]byte, error) {
	s, err := toString(item)
	if nil != err {
		logger.Panicf("invalid item enumeration: %d", item)
	}
	return s, nil
}
