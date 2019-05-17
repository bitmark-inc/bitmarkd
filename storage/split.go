// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"encoding/binary"

	"github.com/bitmark-inc/logger"
)

// PoolNB - handle for a storage pool
type PoolNB struct {
	pool *PoolHandle
}

// Put - store a key/value bytes pair to the database
func (p *PoolNB) Put(key []byte, nValue []byte, bValue []byte) {

	if 8 != len(nValue) {
		logger.Panic("pool.PutNB 1st parameter must be 8 bytes")
		return
	}

	data := make([]byte, len(nValue)+len(bValue))
	copy(data, nValue)
	copy(data[len(nValue):], bValue)
	p.pool.Put(key, data)
}

// Delete - remove a key from the database
func (p *PoolNB) Delete(key []byte) {
	p.pool.Delete(key)
}

// // GetN - read a record and decode first 8 bytes as big endian uint64
// //
// // second parameter is false if record was not found
// // panics if not 8 (or more) bytes in the record
// func (p *PoolNB) GetN(key []byte) (uint64, bool) {
// 	buffer := p.pool.Get(key)
// 	if nil == buffer {
// 		return 0, false
// 	}
// 	if len(buffer) < 8 {
// 		logger.Panicf("pool.GetN truncated record for: %x: %s", key, buffer)
// 	}
// 	n := binary.BigEndian.Uint64(buffer[:8])
// 	return n, true
// }

// GetNB - read a record and decode first 8 bytes as big endian uint64
// and return the rest of the record as byte slice
//
// second parameter is nil if record was not found
// panics if not 9 (or more) bytes in the record
// this returns the actual element in the second parameter - copy the result if it must be preserved
func (p *PoolNB) GetNB(key []byte) (uint64, []byte) {
	buffer := p.pool.Get(key)
	if nil == buffer {
		return 0, nil
	}
	if len(buffer) < 9 { // must have at least one byte after the N value
		logger.Panicf("pool.GetNB truncated record for: %x: %s", key, buffer)
	}
	n := binary.BigEndian.Uint64(buffer[:8])
	return n, buffer[8:]
}

// Has - Check if a key exists
func (p *PoolNB) Has(key []byte) bool {
	return p.pool.Has(key)
}

func (p *PoolNB) BeginDBTransaction() {
	p.pool.Begin()
}

func (p *PoolNB) WriteDBTransaction() {
	p.pool.Commit()
}
