// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/binary"
)

// type to denote a index in the unpaid and available pools
// just a 64 bit unsigned integer - big endian byte order
// (has a Bytes() to feth the big endian representation
type IndexCursor uint64

// holds a cursor for fetching available
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

// convert a count to a byte slice (big endian)
func (ic IndexCursor) Bytes() []byte {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, uint64(ic))
	return buffer
}
