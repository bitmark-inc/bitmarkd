// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package gnomon

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"sync"
	"time"
)

// cursor effectively a limited timestamp
//
// This works like erlang:now() and will advance into future if
// called faster than once per nanosecond continuously.
type Cursor struct {
	seconds     int64
	nanoSeconds int32 // 0 .. 999,999,999
}

// this is to prevent duplicate values
var localData struct {
	sync.RWMutex
	current Cursor
}

// description of binary record
const (
	secondsStart     = 0
	secondsSize      = 8
	nanoSecondsStart = secondsStart + secondsSize
	nanoSecondsSize  = 4
	totalSize        = secondsSize + nanoSecondsSize
	hexSize          = 2 * totalSize
)

// get a current cursor value
//
// ensure that cannot get duplicate value
func NewCursor() *Cursor {

	now := time.Now().UTC()
	cursor := Cursor{
		seconds:     now.Unix(),
		nanoSeconds: int32(now.Nanosecond()),
	}

	localData.Lock()
	defer localData.Unlock()

	if localData.current.seconds > cursor.seconds || localData.current.nanoSeconds >= cursor.nanoSeconds {
		cursor.nanoSeconds = localData.current.nanoSeconds + 1
		cursor.seconds = localData.current.seconds
		if cursor.nanoSeconds > 999999999 {
			cursor.nanoSeconds = 0
			cursor.seconds += 1
		}
	}
	localData.current = cursor
	return &cursor
}

// advance a cursor by one LSB to be the next possible position after
// the its current value
func (cursor *Cursor) Next() {
	cursor.nanoSeconds += 1
	if cursor.nanoSeconds > 999999999 {
		cursor.nanoSeconds = 0
		cursor.seconds += 1
	}
}

// convert to string
func (cursor Cursor) String() string {
	b, err := cursor.MarshalBinary()
	if nil != err {
		fault.Panic("cursor to string failed")
	}
	return hex.EncodeToString(b)
}

// convert to binary
//
// marshal the value in big-endian order (so database indexing will be
// in ascending time order)
func (cursor Cursor) MarshalBinary() ([]byte, error) {
	b := []byte{
		byte(cursor.seconds >> 56), // bytes 1-8: seconds
		byte(cursor.seconds >> 48),
		byte(cursor.seconds >> 40),
		byte(cursor.seconds >> 32),
		byte(cursor.seconds >> 24),
		byte(cursor.seconds >> 16),
		byte(cursor.seconds >> 8),
		byte(cursor.seconds),
		byte(cursor.nanoSeconds >> 24), // bytes 9-12: nanoseconds
		byte(cursor.nanoSeconds >> 16),
		byte(cursor.nanoSeconds >> 8),
		byte(cursor.nanoSeconds),
	}
	return b, nil
}

// convert from binary
func (cursor *Cursor) UnmarshalBinary(s []byte) error {
	if totalSize != len(s) {
		return fault.ErrInvalidLength
	}
	cursor.seconds = int64(s[secondsStart])<<56 |
		int64(s[secondsStart+1])<<48 |
		int64(s[secondsStart+2])<<40 |
		int64(s[secondsStart+3])<<32 |
		int64(s[secondsStart+4])<<24 |
		int64(s[secondsStart+5])<<16 |
		int64(s[secondsStart+6])<<8 |
		int64(s[secondsStart+7])<<0
	cursor.nanoSeconds = int32(s[nanoSecondsStart])<<24 |
		int32(s[nanoSecondsStart+1])<<16 |
		int32(s[nanoSecondsStart+2])<<8 |
		int32(s[nanoSecondsStart+3])<<0

	return nil
}

// convert to JSON
func (cursor Cursor) MarshalJSON() ([]byte, error) {

	b, err := cursor.MarshalBinary()
	if nil != err {
		return nil, err
	}

	// length = '"' + hex characters + '"'
	h := make([]byte, hex.EncodedLen(len(b))+2)
	h[0] = '"'
	hex.Encode(h[1:], b)
	h[len(h)-1] = '"'

	return h, nil
}

// convert from JSON
func (cursor *Cursor) UnmarshalJSON(s []byte) error {

	// special case for null -> same as all '0'
	if 4 == len(s) && "null" == string(s) {
		cursor.seconds = 0
		cursor.nanoSeconds = 0
		return nil
	}

	// length = '"' + hex characters + '"'
	if hexSize+2 != len(s) {
		return fault.ErrInvalidLength
	}
	if '"' != s[0] || '"' != s[len(s)-1] {
		return fault.ErrInvalidCharacter
	}

	b := make([]byte, totalSize)
	_, err := hex.Decode(b, s[1:len(s)-1])
	if nil != err {
		return err
	}

	return cursor.UnmarshalBinary(b)
}
