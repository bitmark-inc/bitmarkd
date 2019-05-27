// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"encoding/binary"

	"github.com/syndtr/goleveldb/leveldb"
	ldb_util "github.com/syndtr/goleveldb/leveldb/util"

	"github.com/bitmark-inc/logger"
)

// TODO: Aaron use interface of Handle
type Handle interface {
	Put(key []byte, value []byte)
	put(key []byte, value []byte)
	PutN(key []byte, value uint64)
	putN(key []byte, value uint64)
	Delete(key []byte)
	remove(key []byte)
	Get(key []byte) []byte
	GetN(key []byte) (uint64, bool)
	getN(key []byte) (uint64, bool)
	GetNB(key []byte) (uint64, []byte)
	getNB(key []byte) (uint64, []byte)
	Has(key []byte) bool
	Begin()
	Commit() error
}

// PoolHandle - the structure of a pool handle
type PoolHandle struct {
	Handle
	prefix     byte
	limit      []byte
	dataAccess DataAccess
}

// Element - a binary data item
type Element struct {
	Key   []byte
	Value []byte
}

// prepend the prefix onto the key
func (p *PoolHandle) prefixKey(key []byte) []byte {
	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	return append(prefixedKey, key...)
}

// Put - store a key/value bytes pair to the database
// TODO: Aaron delete it
func (p *PoolHandle) Put(key []byte, value []byte) {
	poolData.RLock()
	defer poolData.RUnlock()
	if nil == p.dataAccess {
		logger.Panic("pool.Put nil database")
		return
	}
	p.dataAccess.Put(p.prefixKey(key), value)
}

// Put - store a key/value bytes pair to the database
func (p *PoolHandle) put(key []byte, value []byte) {
	poolData.RLock()
	defer poolData.RUnlock()
	if nil == p.dataAccess {
		logger.Panic("pool.Put nil database")
		return
	}
	p.dataAccess.Put(p.prefixKey(key), value)
}

// PutN - store a uint8 as an 8 byte sequence
// TODO: Aaron delete
func (p *PoolHandle) PutN(key []byte, value uint64) {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, value)
	p.Put(key, buffer)
}

// PutN - store a uint8 as an 8 byte sequence
func (p *PoolHandle) putN(key []byte, value uint64) {
	buffer := make([]byte, 8)
	binary.BigEndian.PutUint64(buffer, value)
	p.Put(key, buffer)
}

// Delete - remove a key from the database
// TODO: Aaron delete it
func (p *PoolHandle) Delete(key []byte) {
	poolData.RLock()
	defer poolData.RUnlock()
	p.dataAccess.Delete(p.prefixKey(key))
}

func (p *PoolHandle) remove(key []byte) {
	poolData.RLock()
	defer poolData.RUnlock()
	p.dataAccess.Delete(p.prefixKey(key))
}

// Get - read a value for a given key
//
// this returns the actual element - copy the result if it must be preserved
func (p *PoolHandle) Get(key []byte) []byte {
	poolData.RLock()
	defer poolData.RUnlock()
	if nil == p.dataAccess {
		return nil
	}
	value, err := p.dataAccess.Get(p.prefixKey(key))
	if leveldb.ErrNotFound == err {
		return nil
	}
	logger.PanicIfError("pool.GetB", err)
	return value
}

// GetN - read a record and decode first 8 bytes as big endian uint64
//
// second parameter is false if record was not found
// panics if not 8 (or more) bytes in the record
// TODO: Aaron delete
func (p *PoolHandle) GetN(key []byte) (uint64, bool) {
	buffer := p.Get(key)
	if nil == buffer {
		return 0, false
	}
	if len(buffer) < 8 {
		logger.Panicf("pool.GetN truncated record for: %x: %s", key, buffer)
	}
	n := binary.BigEndian.Uint64(buffer[:8])
	return n, true
}

// GetN - read a record and decode first 8 bytes as big endian uint64
//
// second parameter is false if record was not found
// panics if not 8 (or more) bytes in the record
func (p *PoolHandle) getN(key []byte) (uint64, bool) {
	buffer := p.Get(key)
	if nil == buffer {
		return 0, false
	}
	if len(buffer) < 8 {
		logger.Panicf("pool.GetN truncated record for: %x: %s", key, buffer)
	}
	n := binary.BigEndian.Uint64(buffer[:8])
	return n, true
}

// GetNB - read a record and decode first 8 bytes as big endian uint64
// and return the rest of the record as byte slice
//
// second parameter is nil if record was not found
// panics if not 9 (or more) bytes in the record
// this returns the actual element in the second parameter - copy the result if it must be preserved
// TODO: Aaron delete
func (p *PoolHandle) GetNB(key []byte) (uint64, []byte) {
	buffer := p.Get(key)
	if nil == buffer {
		return 0, nil
	}
	if len(buffer) < 9 { // must have at least one byte after the N value
		logger.Panicf("pool.GetNB truncated record for: %x: %s", key, buffer)
	}
	n := binary.BigEndian.Uint64(buffer[:8])
	return n, buffer[8:]
}

// GetNB - read a record and decode first 8 bytes as big endian uint64
// and return the rest of the record as byte slice
//
// second parameter is nil if record was not found
// panics if not 9 (or more) bytes in the record
// this returns the actual element in the second parameter - copy the result if it must be preserved
func (p *PoolHandle) getNB(key []byte) (uint64, []byte) {
	buffer := p.Get(key)
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
func (p *PoolHandle) Has(key []byte) bool {
	poolData.RLock()
	defer poolData.RUnlock()
	if nil == p.dataAccess {
		return false
	}
	value, err := p.dataAccess.Has(p.prefixKey(key))
	logger.PanicIfError("pool.Has", err)
	return value
}

// LastElement - get the last element in a pool
func (p *PoolHandle) LastElement() (Element, bool) {
	maxRange := ldb_util.Range{
		Start: []byte{p.prefix}, // Start of key range, included in the range
		Limit: p.limit,          // Limit of key range, excluded from the range
	}

	poolData.RLock()
	defer poolData.RUnlock()
	if nil == p.dataAccess {
		return Element{}, false
	}

	iter := p.dataAccess.Iterator(&maxRange)

	found := false
	result := Element{}
	if iter.Last() {

		// contents of the returned slice must not be modified, and are
		// only valid until the next call to Next
		key := iter.Key()
		value := iter.Value()

		dataKey := make([]byte, len(key)-1) // strip the prefix
		copy(dataKey, key[1:])              // ...

		dataValue := make([]byte, len(value))
		copy(dataValue, value)

		result.Key = dataKey
		result.Value = dataValue
		found = true
	}
	iter.Release()
	err := iter.Error()
	logger.PanicIfError("pool.LastElement", err)
	return result, found
}

func (p *PoolHandle) Begin() {
	p.dataAccess.Begin()
}

func (p *PoolHandle) Commit() error {
	return p.dataAccess.Commit()
}
