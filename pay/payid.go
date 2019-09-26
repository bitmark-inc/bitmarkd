// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pay

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// PayId - type to represent a payment identifier
// Note: no reversal is required for this
type PayId [48]byte

// NewPayId - create a payment identifier from a set of transactions
func NewPayId(packed [][]byte) PayId {
	digest := sha3.New384()
	for _, data := range packed {
		digest.Write(data)
	}
	hash := digest.Sum([]byte{})
	var payId PayId
	copy(payId[:], hash)
	return payId
}

// String - convert a binary pay id to hex string for use by the fmt package (for %s)
func (payid PayId) String() string {
	return hex.EncodeToString(payid[:])
}

// GoString - convert a binary pay id to hex string for use by the fmt package (for %#v)
func (payid PayId) GoString() string {
	return "<payid:" + hex.EncodeToString(payid[:]) + ">"
}

// MarshalText - convert pay id to hex text
func (payid PayId) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(payid))
	buffer := make([]byte, size)
	hex.Encode(buffer, payid[:])
	return buffer, nil
}

// UnmarshalText - convert hex text into a pay id
func (payid *PayId) UnmarshalText(s []byte) error {
	if len(*payid) != hex.DecodedLen(len(s)) {
		return fault.NotAPayId
	}
	byteCount, err := hex.Decode(payid[:], s)
	if nil != err {
		return err
	}
	if len(payid) != byteCount {
		return fault.NotAPayId
	}
	return nil
}
