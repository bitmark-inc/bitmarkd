// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"encoding/hex"

	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// PayNonce - type to represent a payment nonce
// Note: no reversal is required for this
type PayNonce [8]byte

// values to round block numbers
const (
	PayNonceHeightMask    uint64 = ^uint64(0x7f)
	PayNonceHeightDelta   uint64 = 0x80
	PayNonceHeightMinimum uint64 = 0xff
)

// NewPayNonce - create a random pay nonce
func NewPayNonce() PayNonce {
	return PayNonceFromBlock(PayNonceRoundedHeight())
}

// get the rounded height
func PayNonceRoundedHeight() uint64 {
	height := blockheader.Height() & PayNonceHeightMask
	if height > PayNonceHeightMinimum {
		height -= PayNonceHeightDelta
	} else {
		height = 1
	}
	return height
}

// PayNonceFromBlock - get a previous paynonce
func PayNonceFromBlock(number uint64) PayNonce {
	nonce := PayNonce{}
	digest, err := blockheader.DigestForBlock(number)
	if nil != err {
		return nonce
	}
	copy(nonce[:], digest[:])
	return nonce
}

// String - convert a binary pay nonce to big endian hex string for use by the fmt package (for %s)
func (paynonce PayNonce) String() string {
	return hex.EncodeToString(paynonce[:])
}

// GoString - convert a binary pay nonce to big endian hex string for use by the fmt package (for %#v)
func (paynonce PayNonce) GoString() string {
	return "<paynonce:" + hex.EncodeToString(paynonce[:]) + ">"
}

// MarshalText - convert pay nonce to big endian hex text
func (paynonce PayNonce) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(paynonce))
	buffer := make([]byte, size)
	hex.Encode(buffer, paynonce[:])
	return buffer, nil
}

// UnmarshalText - convert little endian hex text into a pay nonce
func (paynonce *PayNonce) UnmarshalText(s []byte) error {
	if len(*paynonce) != hex.DecodedLen(len(s)) {
		return fault.NotAPayNonce
	}
	byteCount, err := hex.Decode(paynonce[:], s)
	if nil != err {
		return err
	}
	if len(paynonce) != byteCount {
		return fault.NotAPayNonce
	}
	return nil
}
