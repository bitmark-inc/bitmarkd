// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
)

// bytes in a link, same as merkle hash
const (
	LinkLength = merkle.DigestLength
)

// the type for a link - same as merkle digest
// stored as little endian byte array
// represented as little endian hex text for JSON encoding
type Link merkle.Digest

// Create an link for a packed record
func (record Packed) MakeLink() Link {
	return Link(merkle.NewDigest(record))
}

// convert a binary link to byte slice
func (link Link) Bytes() []byte {
	return link[:]
}

// convert a binary link to little endian hex string for use by the fmt package (for %s)
func (link Link) String() string {
	return hex.EncodeToString(link[:])
}

// convert a binary link to little endian hex string for use by the fmt package (for %#v)
func (link Link) GoString() string {
	return "<link:" + hex.EncodeToString(link[:]) + ">"
}

// convert a little endian hex text representation to a link for use by the format package scan routines
func (link *Link) Scan(state fmt.ScanState, verb rune) error {
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
	if len(token) != hex.EncodedLen(LinkLength) {
		return fault.ErrNotLink
	}

	byteCount, err := hex.Decode(link[:], token)
	if nil != err {
		return err
	}
	if LinkLength != byteCount {
		return fault.ErrNotLink
	}
	return nil
}

// convert link to little endian hex text
func (link Link) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(LinkLength)
	buffer := make([]byte, size)
	hex.Encode(buffer, link[:])
	return buffer, nil
}

// convert little endian hex text into a link
func (link *Link) UnmarshalText(s []byte) error {
	if LinkLength != hex.DecodedLen(len(s)) {
		return fault.ErrNotLink
	}
	byteCount, err := hex.Decode(link[:], s)
	if nil != err {
		return err
	}
	if LinkLength != byteCount {
		return fault.ErrNotLink
	}
	return nil
}

// convert and validate little endian binary byte slice to a link
func LinkFromBytes(link *Link, buffer []byte) error {
	if LinkLength != len(buffer) {
		return fault.ErrNotLink
	}
	copy(link[:], buffer)
	return nil
}
