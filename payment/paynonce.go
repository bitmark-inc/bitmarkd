// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"hash/crc64"
)

// type to represent a payment nonce
// Note: no reversal is required for this
type PayNonce [8]byte

var table = crc64.MakeTable(crc64.ECMA)

// create a random pay nonce
func NewPayNonce() PayNonce {

	crc := block.GetLatestCRC()

	nonce := PayNonce{}
	binary.BigEndian.PutUint64(nonce[:], crc)
	return nonce
}

// convert a binary pay nonce to big endian hex string for use by the fmt package (for %s)
func (paynonce PayNonce) String() string {
	return hex.EncodeToString(paynonce[:])
}

// convert a binary pay nonce to big endian hex string for use by the fmt package (for %#v)
func (paynonce PayNonce) GoString() string {
	return "<paynonce:" + hex.EncodeToString(paynonce[:]) + ">"
}

// convert pay nonce to big endian hex text
func (paynonce PayNonce) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(paynonce))
	buffer := make([]byte, size)
	hex.Encode(buffer, paynonce[:])
	return buffer, nil
}

// convert little endian hex text into a pay nonce
func (paynonce *PayNonce) UnmarshalText(s []byte) error {
	if len(*paynonce) != hex.DecodedLen(len(s)) {
		return fault.ErrNotAPayNonce
	}
	byteCount, err := hex.Decode(paynonce[:], s)
	if nil != err {
		return err
	}
	if len(paynonce) != byteCount {
		return fault.ErrNotAPayNonce
	}
	return nil
}
