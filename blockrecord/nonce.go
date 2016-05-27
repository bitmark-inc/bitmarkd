// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// type for nonce
type NonceType uint64

// convert a nonce to little endian hex for JSON
func (nonce NonceType) MarshalJSON() ([]byte, error) {

	bits := make([]byte, 8)
	binary.LittleEndian.PutUint64(bits, uint64(nonce))

	size := 2 + hex.EncodedLen(len(bits))
	buffer := make([]byte, size)
	buffer[0] = '"'
	buffer[size-1] = '"'
	hex.Encode(buffer[1:], bits)
	return buffer, nil
}

// convert a nonce little endian hex string to nonce value
func (nonce *NonceType) UnmarshalJSON(s []byte) error {
	// length = '"' + characters + '"'
	last := len(s) - 1
	if '"' != s[0] || '"' != s[last] {
		return fault.ErrInvalidCharacter
	}

	b := s[1:last]

	buffer := make([]byte, hex.DecodedLen(len(b)))
	_, err := hex.Decode(buffer, b)
	if nil != err {
		return err
	}
	*nonce = NonceType(binary.LittleEndian.Uint64(buffer))
	return nil
}
