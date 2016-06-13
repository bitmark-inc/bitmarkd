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

// a prefix for the client (for Bitcoin submission)
const (
	linkPrefix       = "BMK1"
	linkPrefixLength = len(linkPrefix)
	LinkLength       = merkle.DigestLength
)

// the type for a link - same as merkle digest
// stored as little endian byte array
// represented as big endian hex value for print
// represented as little endian hex text for JSON encoding
type Link merkle.Digest

// Create an link for a packed record
func (record Packed) MakeLink() Link {
	return Link(merkle.NewDigest(record))
}

// internal function to return a reversed byte order copy of a link
func (d Link) reversed() []byte {
	result := make([]byte, LinkLength)
	for i := 0; i < LinkLength; i += 1 {
		result[i] = d[LinkLength-1-i]
	}
	return result
}

// internal function to return a reversed byte order copy of a link
func (link *Link) reversedFromBytes(buffer []byte) {
	for i := 0; i < LinkLength; i += 1 {
		link[i] = buffer[LinkLength-1-i]
	}
}

// convert a binary link to byte slice
func (link Link) Bytes() []byte {
	return link[:]
}

// convert a binary link to big endian hex string for use by the fmt package (for %s)
func (link Link) String() string {
	return hex.EncodeToString(link.reversed())
}

// convert a binary link to big endian hex string for use by the fmt package (for %#v)
func (link Link) GoString() string {
	return "<link:" + hex.EncodeToString(link.reversed()) + ">"
}

// convert a big endian hex text representation to a link for use by the format package scan routines
func (link *Link) Scan(state fmt.ScanState, verb rune) error {
	count := 0
	token, err := state.Token(true, func(c rune) bool {
		if count < linkPrefixLength { // allow anything for prefix - exact check later
			count += 1
			return true
		}
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
	expectedSize := linkPrefixLength + hex.EncodedLen(LinkLength)
	actualSize := len(token)
	if expectedSize != actualSize {
		return fault.ErrNotLink
	}

	if linkPrefix != string(token[:linkPrefixLength]) {
		return fault.ErrNotLink
	}

	buffer := make([]byte, LinkLength)
	byteCount, err := hex.Decode(buffer, token[linkPrefixLength:])
	if nil != err {
		return err
	}

	if LinkLength != byteCount {
		return fault.ErrNotLink
	}
	link.reversedFromBytes(buffer)
	return nil
}

// convert link to little endian hex text
func (link Link) MarshalText() ([]byte, error) {
	// stage the prefix and link
	stageSize := linkPrefixLength + len(link)
	stage := make([]byte, stageSize)
	copy(stage, linkPrefix)

	for i := 0; i < LinkLength; i += 1 {
		stage[linkPrefixLength+i] = link[i]
	}

	// encode to hex
	size := hex.EncodedLen(stageSize)
	buffer := make([]byte, size)
	hex.Encode(buffer, stage)
	return buffer, nil
}

// convert little endian hex text into a link
func (link *Link) UnmarshalText(s []byte) error {
	buffer := make([]byte, hex.DecodedLen(len(s)))
	byteCount, err := hex.Decode(buffer, s)
	if nil != err {
		return err
	}
	if linkPrefixLength+LinkLength != byteCount {
		return fault.ErrNotLink
	}
	if linkPrefix != string(buffer[:linkPrefixLength]) {
		return fault.ErrNotLink
	}

	for i := 0; i < LinkLength; i += 1 {
		link[i] = buffer[linkPrefixLength+i]
	}
	return nil
}

// convert and validate little endian binary byte slice to a link
func LinkFromBytes(link *Link, buffer []byte) error {
	if LinkLength != len(buffer) {
		return fault.ErrNotLink
	}
	for i := 0; i < LinkLength; i += 1 {
		link[i] = buffer[i]
	}
	return nil
}

// convert and validate a little endian hex link string
// Notes:
// 1. hex code contains prefix at the beginning
// 2. this is hex code in same order as hex JSON above
func LinkFromHexString(link *Link, hexWithPrefix string) error {

	if len(hexWithPrefix) != 2*(linkPrefixLength+LinkLength) {
		return fault.ErrNotLink
	}

	buffer := make([]byte, linkPrefixLength+LinkLength)
	byteCount, err := hex.Decode(buffer, []byte(hexWithPrefix))
	if nil != err {
		return err
	}

	if linkPrefixLength+LinkLength != byteCount {
		return fault.ErrNotLink
	}
	if string(buffer[0:linkPrefixLength]) != linkPrefix {
		return fault.ErrNotLink
	}

	return LinkFromBytes(link, buffer[linkPrefixLength:])
}
