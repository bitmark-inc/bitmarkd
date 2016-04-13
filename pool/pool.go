// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"math/big"
	"sync"
)

// holds the database handle
var poolData struct {
	sync.Mutex
	database *leveldb.DB
}

// the pool handle
type Pool struct {
	sync.RWMutex
	prefix byte
	limit  []byte
}

// a binary data item
type Element struct {
	Key   []byte
	Value []byte
}

// for database version
var versionKey = []byte{internalPrefix, 'V', 'E', 'R', 'S', 'I', 'O', 'N'}
var currentVersion = []byte{0x00, 0x00, 0x00, 0x01}

// open up the database connection
//
// this must be called before any pool.New() is created
func Initialise(database string) {
	poolData.Lock()
	defer poolData.Unlock()

	if nil != poolData.database {
		fault.Panic("pool.Initialise - already done")
	}

	db, err := leveldb.RecoverFile(database, nil)
	// db, err := leveldb.OpenFile(database, nil)

	fault.PanicIfError("pool.Initialise", err)

	poolData.database = db

	// ensure that the database is compatible
	versionValue, err := poolData.database.Get(versionKey, nil)
	if leveldb.ErrNotFound == err {
		err := poolData.database.Put(versionKey, currentVersion, nil)
		fault.PanicIfError("pool.Initialise set version", err)
	} else if nil != err {
		fault.PanicWithError("pool.Initialise get version", err)
	} else if !bytes.Equal(versionValue, currentVersion) {
		fault.Panicf("incompatible database version: expected: %x  actual: %x", currentVersion, versionValue)
	}
}

// close the database connection
func Finalise() {
	poolData.Lock()
	defer poolData.Unlock()

	// no need to stop if already stopped
	if nil == poolData.database {
		return
	}

	poolData.database.Close()
	poolData.database = nil
	return
}

// create a new pool with a specific key prefix
func New(prefix nameb) *Pool {
	poolData.Lock()
	defer poolData.Unlock()
	if nil == poolData.database {
		fault.Panic("pool.New - not initialised")
	}
	limit := []byte(nil)
	if prefix < 255 {
		limit = []byte{byte(prefix) + 1}
	}
	pool := Pool{
		prefix: byte(prefix),
		limit:  limit,
	}

	return &pool
}

// flush the pool channel
func (p *Pool) Flush() {
	// p.Lock()
	// defer p.Unlock()
	// not needed
}

// prepend the prefix onto the key
func (p *Pool) prefixKey(key []byte) []byte {
	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	return append(prefixedKey, key...)
}

// add a key/value bytes pair to the database
func (p *Pool) Add(key []byte, value []byte) {
	p.Lock()
	defer p.Unlock()
	err := poolData.database.Put(p.prefixKey(key), value, nil)
	fault.PanicIfError("pool.Add", err)
}

// remove a key from the database
func (p *Pool) Remove(key []byte) {
	p.Lock()
	defer p.Unlock()
	err := poolData.database.Delete(p.prefixKey(key), nil)
	fault.PanicIfError("pool.Remove", err)
}

// read a value for a given key
//
// this returns the actual element - copy the result if it must be preserved
func (p *Pool) Get(key []byte) []byte {
	p.Lock()
	defer p.Unlock()
	value, err := poolData.database.Get(p.prefixKey(key), nil)
	if leveldb.ErrNotFound == err {
		return nil
	}
	fault.PanicIfError("pool.Get", err)
	return value
}

// Check if a key exists
func (p *Pool) Has(key []byte) bool {
	p.Lock()
	defer p.Unlock()
	value, err := poolData.database.Has(p.prefixKey(key), nil)
	fault.PanicIfError("pool.Has", err)
	return value
}

// get the last element in a pool
func (p *Pool) LastElement() (Element, bool) {
	maxRange := util.Range{
		Start: []byte{p.prefix}, // Start of key range, included in the range
		Limit: p.limit,          // Limit of key range, excluded from the range
	}

	iter := poolData.database.NewIterator(&maxRange, nil)

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
	fault.PanicIfError("pool.LastElement", err)
	return result, found
}

// cursor structure
type FetchCursor struct {
	pool     *Pool
	maxRange util.Range
}

// initialise a cursor to the start of a key range
func (p *Pool) NewFetchCursor() *FetchCursor {

	return &FetchCursor{
		pool: p,
		maxRange: util.Range{
			Start: []byte{p.prefix}, // Start of key range, included in the range
			Limit: p.limit,          // Limit of key range, excluded from the range
		},
	}
}

func (cursor *FetchCursor) Seek(key []byte) *FetchCursor {
	cursor.maxRange.Start = cursor.pool.prefixKey(key)
	return cursor
}

// to increment the key
var one = big.NewInt(1)

// fetch some elements starting from key
func (cursor *FetchCursor) Fetch(count int) ([]Element, error) {
	if nil == cursor {
		return nil, fault.ErrInvalidCursor
	}
	if count <= 0 {
		return nil, fault.ErrInvalidCount
	}

	iter := poolData.database.NewIterator(&cursor.maxRange, nil)

	results := make([]Element, 0, count)
	n := 0
	for iter.Next() {

		// contents of the returned slice must not be modified, and are
		// only valid until the next call to Next
		key := iter.Key()
		value := iter.Value()

		dataKey := make([]byte, len(key)-1) // strip the prefix
		copy(dataKey, key[1:])              // ...

		dataValue := make([]byte, len(value))
		copy(dataValue, value)

		e := Element{
			Key:   dataKey,
			Value: dataValue,
		}
		results = append(results, e)
		n += 1
		if n >= count {
			break
		}
	}
	iter.Release()
	err := iter.Error()

	if n > 0 {
		keyLen := len(results[n-1].Key)
		if len(cursor.maxRange.Start) != keyLen+1 {
			cursor.maxRange.Start = make([]byte, keyLen+1)
		}
		cursor.maxRange.Start[0] = cursor.pool.prefix
		b := big.Int{}
		copy(cursor.maxRange.Start[1:], b.SetBytes(results[n-1].Key).Add(&b, one).Bytes())
	}
	return results, err
}

type Iterator struct {
	iter iterator.Iterator
}

// fetch some elements starting from key
func (p *Pool) Iterate(key []byte) *Iterator {

	maxRange := util.Range{
		Start: p.prefixKey(key), // Start of key range, included in the range
		Limit: p.limit,          // Limit of key range, excluded from the range
	}

	return &Iterator{
		iter: poolData.database.NewIterator(&maxRange, nil),
	}
}

func (it *Iterator) Next() *Element {
	if !it.iter.Next() {
		return nil
	}

	// contents of the returned slice must not be modified, and are
	// only valid until the next call to Next
	key := it.iter.Key()
	value := it.iter.Value()

	dataKey := make([]byte, len(key)-1) // strip the prefix
	copy(dataKey, key[1:])              // ...

	dataValue := make([]byte, len(value))
	copy(dataValue, value)

	return &Element{
		Key:   dataKey,
		Value: dataValue,
	}
}

// must release the iterator when finished with it
func (it *Iterator) Release() {
	it.iter.Release()
	err := it.iter.Error()
	fault.PanicIfError("pool.Iterator.Release", err)
}
