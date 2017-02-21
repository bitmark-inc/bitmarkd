// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord

import (
	"encoding/binary"
	"encoding/hex"
)

// type for nonce
type NonceType uint64

// convert a nonce to little endian hex for JSON
func (nonce NonceType) MarshalText() ([]byte, error) {

	bits := make([]byte, 8)
	binary.LittleEndian.PutUint64(bits, uint64(nonce))

	size := hex.EncodedLen(len(bits))
	buffer := make([]byte, size)
	hex.Encode(buffer, bits)
	return buffer, nil
}

// convert a nonce little endian hex string to nonce value
func (nonce *NonceType) UnmarshalText(b []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(b)))
	_, err := hex.Decode(buffer, b)
	if nil != err {
		return err
	}
	*nonce = NonceType(binary.LittleEndian.Uint64(buffer))
	return nil
}
