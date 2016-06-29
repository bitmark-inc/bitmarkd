// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"golang.org/x/crypto/sha3"
)

// type to represent a payment identifier
// this is considered as a big endian value for difficulty comparison
// Note: no reversal is required for this
type PayId [48]byte

// create a payment identifier from a set of transactions
func NewPayId(packed []byte) PayId {
	return sha3.Sum384(packed)
}

// convert a binary pay id to big endian hex string for use by the fmt package (for %s)
func (payid PayId) String() string {
	return hex.EncodeToString(payid[:])
}

// convert a binary pay id to big endian hex string for use by the fmt package (for %#v)
func (payid PayId) GoString() string {
	return "<payid:" + hex.EncodeToString(payid[:]) + ">"
}

// convert pay id to big endian hex text
func (payid PayId) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(payid))
	buffer := make([]byte, size)
	hex.Encode(buffer, payid[:])
	return buffer, nil
}

// convert little endian hex text into a pay id
func (payid *PayId) UnmarshalText(s []byte) error {
	if len(*payid) != hex.DecodedLen(len(s)) {
		return fault.ErrNotAPayId
	}
	byteCount, err := hex.Decode(payid[:], s)
	if nil != err {
		return err
	}
	if len(payid) != byteCount {
		return fault.ErrNotAPayId
	}
	return nil
}
