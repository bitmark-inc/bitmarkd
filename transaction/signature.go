// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// the type for a signature
type Signature []byte

// convert a binary signature to hex string for use by the fmt package (for %s)
func (signature Signature) String() string {
	return hex.EncodeToString(signature)
}

// convert a binary signature to hex string for use by the fmt package (for %#v)
func (signature Signature) GoString() string {
	return "<signature:" + hex.EncodeToString(signature) + ">"
}

// convert a text representation to a signature for use by the format package scan routines
func (signature *Signature) Scan(state fmt.ScanState, verb rune) error {
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
	sig := make([]byte, hex.DecodedLen(len(token)))
	byteCount, err := hex.Decode(sig, token)
	if nil != err {
		return err
	}
	*signature = sig[:byteCount]
	return nil
}

// convert a binary signature to hex for unpacking record to JSON
func (signature Signature) MarshalJSON() ([]byte, error) {
	size := 2 + hex.EncodedLen(len(signature))
	b := make([]byte, size)
	b[0] = '"'
	b[size-1] = '"'
	hex.Encode(b[1:], signature)
	return b, nil
}

// convert an hex string to a signature for conversion from JSON
func (signature *Signature) UnmarshalJSON(s []byte) error {
	// length = '"' + hex characters + '"'
	last := len(s) - 1
	if '"' != s[0] || '"' != s[last] {
		return fault.ErrInvalidCharacter
	}
	return signature.UnmarshalText(s[1:last])
}

// convert signature to text
func (signature Signature) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(signature))
	b := make([]byte, size)
	hex.Encode(b, signature)
	return b, nil
}

// convert text into a signature
func (signature *Signature) UnmarshalText(s []byte) error {
	sig := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(sig, s)
	if nil != err {
		return err
	}
	*signature = sig[:byteCount]
	return nil
}
