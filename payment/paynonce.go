// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// type to represent a payment nonce
type PayNonce [8]byte

// create a random pay nonce
func NewPayNonce() PayNonce {

	buffer := make([]byte, 8)
	_, err := rand.Read(buffer)
	fault.PanicIfError("rand.Read failed", err)

	nonce := PayNonce{}
	copy(nonce[:], buffer)
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

// convert pay nonce to little endian hex text
func (paynonce PayNonce) MarshalText() ([]byte, error) {
	// encode to hex
	size := hex.EncodedLen(len(paynonce))
	buffer := make([]byte, size)
	hex.Encode(buffer, paynonce[:])
	return buffer, nil
}

// convert little endian hex text into a pay nonce
func (paynonce *PayNonce) UnmarshalText(s []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(buffer, s)
	if nil != err {
		return err
	}
	if len(paynonce) != byteCount {
		return fault.ErrNotAPayNonce
	}
	for i := 0; i < len(paynonce); i += 1 {
		paynonce[i] = buffer[i]
	}
	return nil
}