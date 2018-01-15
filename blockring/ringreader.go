// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockring

// to iterate though the ring
type RingReader struct {
	stop    int
	current int
	crc     uint64
}

// start of ring iterator
func NewRingReader() *RingReader {
	globalData.Lock()
	i := globalData.ringIndex
	globalData.Unlock()

	c := i - 1
	if c < 0 {
		c = len(globalData.ring) - 1
	}
	r := &RingReader{
		stop:    i,
		current: c,
	}
	return r
}

// fetch item from ring
// works in reverse, fetching older items
func (r *RingReader) Next() bool {
	if r.stop == r.current {
		return false
	}
	r.crc = globalData.ring[r.current].crc
	r.current -= 1
	if r.current < 0 {
		r.current = len(globalData.ring) - 1
	}
	return true
}

// read the fetched value
func (r *RingReader) GetCRC() uint64 {
	return r.crc
}
