// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package merkle

import (
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"golang.org/x/crypto/sha3"
)

// number of bytes in the digest
const DigestLength = 32

// type for a digest
// stored as little endian byte array
// represented as big endian hex value for print
// represented as little endian hex text for JSON encoding
// to convert to bytes just use d[:]
type Digest [DigestLength]byte

// create a digest from a byte slice
func NewDigest(record []byte) Digest {
	return sha3.Sum256(record)
}

// internal function to return a reversed byte order copy of a digest
func reversed(d Digest) []byte {
	result := make([]byte, DigestLength)
	for i := 0; i < DigestLength; i += 1 {
		result[i] = d[DigestLength-1-i]
	}
	return result
}

// convert a binary digest to hex string for use by the fmt package (for %s)
//
// the stored version is in little endian, but the output string is big endian
func (digest Digest) String() string {
	return hex.EncodeToString(reversed(digest))
}

// convert a binary digest to big endian hex string for use by the fmt package (for %#v)
func (digest Digest) GoString() string {
	return "<SHA3-256:" + hex.EncodeToString(reversed(digest)) + ">"
}

// convert a big endian hex representation to a digest for use by the format package scan routines
func (digest *Digest) Scan(state fmt.ScanState, verb rune) error {
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
	if len(token) != hex.EncodedLen(DigestLength) {
		return fault.ErrNotLink
	}

	buffer := make([]byte, hex.DecodedLen(len(token)))
	byteCount, err := hex.Decode(buffer, token)
	if nil != err {
		return err
	}

	for i, v := range buffer[:byteCount] {
		digest[DigestLength-1-i] = v
	}
	return nil
}

// convert digest to little endian hex text
func (digest Digest) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(digest))
	buffer := make([]byte, size)
	hex.Encode(buffer, digest[:])
	return buffer, nil
}

// convert little endian hex text into a digest
func (digest *Digest) UnmarshalText(s []byte) error {
	if DigestLength != hex.DecodedLen(len(s)) {
		return fault.ErrNotLink
	}
	// byteCount, err := hex.Decode(digest[:], s)
	// if nil != err {
	// 	return err
	// }
	// if DigestLength != byteCount {
	// 	return fault.ErrNotLink
	// }
	// return nil

	buffer := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(buffer, s)
	if nil != err {
		return err
	}
	for i, v := range buffer[:byteCount] {
		digest[i] = v
	}
	return nil
}

// convert and validate little endian binary byte slice to a digest
func DigestFromBytes(digest *Digest, buffer []byte) error {
	if DigestLength != len(buffer) {
		return fault.ErrNotLink
	}
	copy(digest[:], buffer)
	return nil
}
