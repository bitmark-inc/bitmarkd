// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// a prefix for the client (for Bitcoin submission)
const (
	linkPrefix     = "BMK0"
	linkPrefixSize = len(linkPrefix)
	LinkSize       = block.DigestSize
)

// the type for a link - same as block digest
// stored as little endian byte array
// represented as big endian hex value for print
// represented as little endian hex text for JSON encoding
type Link block.Digest

// Create an link for a packed record
//
// reuse the block algorithm
func (record Packed) MakeLink() Link {
	return Link(block.NewDigest(record))
}

// internal function to return a reversed byte order copy of a link
func (d Link) reversed() []byte {
	result := make([]byte, LinkSize)
	for i := 0; i < LinkSize; i += 1 {
		result[i] = d[LinkSize-1-i]
	}
	return result
}

// internal function to return a reversed byte order copy of a link
func (link *Link) reversedFromBytes(buffer []byte) {
	for i := 0; i < LinkSize; i += 1 {
		link[i] = buffer[LinkSize-1-i]
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
		if count < linkPrefixSize { // allow anything for prefix - exact check later
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
	expectedSize := linkPrefixSize + hex.EncodedLen(LinkSize)
	actualSize := len(token)
	if expectedSize != actualSize {
		return fault.ErrNotLink
	}

	if linkPrefix != string(token[:linkPrefixSize]) {
		return fault.ErrNotLink
	}

	buffer := make([]byte, LinkSize)
	byteCount, err := hex.Decode(buffer, token[linkPrefixSize:])
	if nil != err {
		return err
	}

	if LinkSize != byteCount {
		return fault.ErrNotLink
	}
	link.reversedFromBytes(buffer)
	return nil
}

// convert a binary link to little endian hex text for JSON
//
// ***** possibly re-use MarshalText to save code duplication
// ***** but would that cost more buffer copying?
func (link Link) MarshalJSON() ([]byte, error) {
	// stage the prefix and link
	stageSize := linkPrefixSize + len(link)
	stage := make([]byte, stageSize)
	copy(stage, linkPrefix)

	for i := 0; i < LinkSize; i += 1 {
		stage[linkPrefixSize+i] = link[i]
	}

	// encode to hex
	size := 2 + hex.EncodedLen(stageSize)
	buffer := make([]byte, size)
	buffer[0] = '"'
	buffer[size-1] = '"'
	hex.Encode(buffer[1:], stage)
	return buffer, nil
}

// convert a little endian hex string to a link for conversion from JSON
func (link *Link) UnmarshalJSON(s []byte) error {
	// length = '"' + characters + '"'
	last := len(s) - 1
	if '"' != s[0] || '"' != s[last] {
		return fault.ErrInvalidCharacter
	}
	return link.UnmarshalText(s[1:last])
}

// convert link to little endian hex text
//
// ***** possibly use NewEncoder and byte buffer to save copy
func (link Link) MarshalText() ([]byte, error) {
	// stage the prefix and link
	stageSize := linkPrefixSize + len(link)
	stage := make([]byte, stageSize)
	copy(stage, linkPrefix)

	for i := 0; i < LinkSize; i += 1 {
		stage[linkPrefixSize+i] = link[i]
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
	if linkPrefixSize+LinkSize != byteCount {
		return fault.ErrNotLink
	}
	if linkPrefix != string(buffer[:linkPrefixSize]) {
		return fault.ErrNotLink
	}

	for i := 0; i < LinkSize; i += 1 {
		link[i] = buffer[linkPrefixSize+i]
	}
	return nil
}

// convert and validate little endian binary byte slice to a link
func LinkFromBytes(link *Link, buffer []byte) error {
	if LinkSize != len(buffer) {
		return fault.ErrNotLink
	}
	for i := 0; i < LinkSize; i += 1 {
		link[i] = buffer[i]
	}
	return nil
}

// convert and validate a little endian hex link string
// Notes:
// 1. hex code contains prefix at the beginning
// 2. this is hex code in same order as hex JSON above
func LinkFromHexString(link *Link, hexWithPrefix string) error {

	buffer := make([]byte, linkPrefixSize+LinkSize)
	byteCount, err := hex.Decode(buffer, []byte(hexWithPrefix))
	if nil != err {
		return err
	}

	if linkPrefixSize+LinkSize != byteCount {
		return fault.ErrNotLink
	}
	if string(buffer[0:linkPrefixSize]) != linkPrefix {
		return fault.ErrNotLink
	}

	return LinkFromBytes(link, buffer[linkPrefixSize:])
}
