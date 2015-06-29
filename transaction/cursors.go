// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"sync/atomic"
)

// type to denote a index in the unpaid, pending or confirmed pools
// just a 64 bit unsigned integer - big endian byte order
// (has a Bytes() to fetch the big endian representation
type IndexCursor uint64

// holds a cursor for fetching confirmed with associated assets
type AvailableCursor struct {
	count  IndexCursor
	assets map[Link]struct{}
}

// create a new cursor for FetchAvailable
func NewAvailableCursor() *AvailableCursor {
	return &AvailableCursor{
		count:  0,
		assets: make(map[Link]struct{}),
	}
}

const (
	cursorByteLength = 8  // because of underlying uint64
)

// convert a count to a byte slice (big endian)
func (ic IndexCursor) Bytes() []byte {
	buffer := make([]byte, cursorByteLength)
	binary.BigEndian.PutUint64(buffer, uint64(ic))
	return buffer
}

// convert a next count to a byte slice (big endian)
func (ic *IndexCursor) NextBytes() []byte {

	// avoid needing a mutex lock
	nextValue := atomic.AddUint64((*uint64)(ic), 1)

	buffer := make([]byte, cursorByteLength)
	binary.BigEndian.PutUint64(buffer,nextValue)
	return buffer
}

// convert to string
func (ic IndexCursor) String() string {
	return fmt.Sprintf("IC:%08x", uint64(ic))
}

// convert link to little endian base64 text
func (ic IndexCursor) MarshalText() ([]byte, error) {
	buffer := make([]byte, cursorByteLength)
	binary.BigEndian.PutUint64(buffer, uint64(ic))

	stage := make([]byte, base64.StdEncoding.EncodedLen(cursorByteLength))

	base64.StdEncoding.Encode(stage, buffer)
	return stage, nil
}

// convert little endian base64 text into a link
func (ic *IndexCursor) UnmarshalText(s []byte) error {

	buffer := make([]byte, base64.StdEncoding.DecodedLen(len(s)))

	byteCount, err := base64.StdEncoding.Decode(buffer, s)
	if nil != err {
		return err
	}

	if byteCount != cursorByteLength {
		return fault.ErrInvalidLength
	}

	*ic = IndexCursor(binary.BigEndian.Uint64(buffer))
	return nil
}

// convert to JSON
func (ic IndexCursor) MarshalJSON() ([]byte, error) {

	b, err := ic.MarshalText()
	if nil != err {
		return nil, err
	}

	// length = '"' + characters + '"'
	s := make([]byte, len(b) + 2)
	s[0] = '"'
	copy(s[1:], b)
	s[len(s)-1] = '"'

	return s, nil
}

// convert from JSON
func (ic *IndexCursor) UnmarshalJSON(s []byte) error {

	// special case for null -> same as all '0'
	if 4 == len(s) && "null" == string(s) {
		*ic = 0
		return nil
	}

	if '"' != s[0] || '"' != s[len(s)-1] {
		return fault.ErrInvalidCharacter
	}

	return ic.UnmarshalText(s[1:len(s)-1])
}
