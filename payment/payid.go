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

// convert pay id to little endian hex text
func (payid PayId) MarshalText() ([]byte, error) {
	// encode to hex
	size := hex.EncodedLen(len(payid))
	buffer := make([]byte, size)
	hex.Encode(buffer, payid[:])
	return buffer, nil
}

// convert little endian hex text into a pay id
func (payid *PayId) UnmarshalText(s []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(buffer, s)
	if nil != err {
		return err
	}
	if len(payid) != byteCount {
		return fault.ErrNotAPayId
	}
	for i := 0; i < len(payid); i += 1 {
		payid[i] = buffer[i]
	}
	return nil
}
