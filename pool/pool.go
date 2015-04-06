// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"container/list"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
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
	prefix    byte
	cacheSize int
	lru       list.List
	index     map[string]*list.Element
}

// a binary data item
type Element struct {
	Key   []byte
	Value []byte
}

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

// create a new pool with a specific key prefix an optional local memory cache
//
// A separate index is created to provide a timestamp ordering
func New(prefix nameb, cacheSize int) *Pool {
	poolData.Lock()
	defer poolData.Unlock()
	if nil == poolData.database {
		fault.Panic("pool.New - not initialised")
	}
	pool := Pool{
		prefix:    byte(prefix),
		cacheSize: cacheSize,
		lru:       list.List{},
		index:     make(map[string]*list.Element),
	}

	return &pool
}

// add a key/value bytes pair to the database
func (p *Pool) Add(key []byte, value []byte) {
	p.Lock()
	defer p.Unlock()

	if p.cacheSize > 0 {
		stringKey := string(key)
		// if item in LRU the move to front
		if element, ok := p.index[stringKey]; ok {

			p.lru.MoveToFront(element)
			e := element.Value.(*Element)

			valueLen := len(value)
			if cap(e.Value) < valueLen {
				e.Value = make([]byte, valueLen)
			}
			copy(e.Value[:valueLen], value) // ensure all data is copied even if old data was shorter
			e.Value = e.Value[:valueLen]    // truncate in case the new data is shorter than old data

		} else if p.lru.Len() >= p.cacheSize {

			// not in LRU - need to re-use element
			element := p.lru.Back()
			e := element.Value.(*Element)
			delete(p.index, string(e.Key))

			keyLen := len(key)
			if cap(e.Key) < keyLen {
				e.Key = make([]byte, keyLen)
			}
			copy(e.Key[:keyLen], key) // ensure all data is copied even if old data was shorter
			e.Key = e.Key[:keyLen]    // truncate in case the new data is shorter than old data

			valueLen := len(value)
			if cap(e.Value) < valueLen {
				e.Value = make([]byte, valueLen)
			}
			copy(e.Value[:valueLen], value) // ensure all data is copied even if old data was shorter
			e.Value = e.Value[:valueLen]    // truncate in case the new data is shorter than old data

			p.lru.MoveToFront(element)
			p.index[stringKey] = element

		} else {

			// not in LRU - have space to add new entry
			k := make([]byte, len(key))
			v := make([]byte, len(value))
			copy(k, key)
			copy(v, value)
			element := Element{
				Key:   k,
				Value: v,
			}
			p.index[stringKey] = p.lru.PushFront(&element)
		}
	}

	// write to database
	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	prefixedKey = append(prefixedKey, key...)

	err := poolData.database.Put(prefixedKey, value, nil)
	fault.PanicIfError("pool.Add", err)

}

// remove a key from the database
func (p *Pool) Remove(key []byte) {
	p.Lock()
	defer p.Unlock()

	if p.cacheSize > 0 {
		stringKey := string(key)
		// if item in LRU the move to front
		if element, ok := p.index[stringKey]; ok {
			p.lru.Remove(element)
			delete(p.index, stringKey)
		}
	}

	// delete from database
	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	prefixedKey = append(prefixedKey, key...)

	err := poolData.database.Delete(prefixedKey, nil)
	fault.PanicIfError("pool.Remove", err)
}

// read a value for a given key
//
// this returns the actual element - copy it if you need to
func (p *Pool) Get(key []byte) ([]byte, bool) {
	p.Lock()
	defer p.Unlock()

	stringKey := string(key)
	if element, ok := p.index[stringKey]; ok {
		p.lru.MoveToFront(element)
		return element.Value.(*Element).Value, true
	}

	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	prefixedKey = append(prefixedKey, key...)
	value, err := poolData.database.Get(prefixedKey, nil)
	if leveldb.ErrNotFound == err {
		return nil, false
	}
	fault.PanicIfError("pool.Get", err)

	return value, true
}

// get the last element in a pool
func (p *Pool) LastElement() (Element, bool) {

	maxRange := util.Range{
		Start: []byte{p.prefix},     // Start of key range, included in the range
		Limit: []byte{p.prefix + 1}, // Limit of key range, excluded from the range
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

// fetch some elements starting from key
func (p *Pool) Fetch(key []byte, count int) ([]Element, error) {
	if count <= 0 {
		return nil, fault.ErrInvalidCount
	}

	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	prefixedKey = append(prefixedKey, key...)

	maxRange := util.Range{
		Start: prefixedKey,          // Start of key range, included in the range
		Limit: []byte{p.prefix + 1}, // Limit of key range, excluded from the range
	}

	iter := poolData.database.NewIterator(&maxRange, nil)

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
	return results, err
}

// // fetch a range of elements starting from key prefix
// //
// // fetch all records with the same prefix
// func (p *Pool) FetchPrefixed(key []byte) []Element {

// 	length := len(key) + 1
// 	prefixedStart := make([]byte, 1, length)
// 	prefixedStart[0] = p.prefix
// 	prefixedStart = append(prefixedStart, key...)

// 	prefixedFinish := make([]byte, length)
// 	copy(prefixedFinish, prefixedStart)

// loop:
// 	for i := length - 1; i >= 0; i -= 1 {
// 		prefixedFinish[i] += 1
// 		if 0 != prefixedFinish[i] {
// 			break loop
// 		}
// 	}

// 	maxRange := util.Range{
// 		Start: prefixedStart,  // Start of key range, included in the range
// 		Limit: prefixedFinish, // Limit of key range, excluded from the range
// 	}

// 	iter := poolData.database.NewIterator(&maxRange, nil)

// 	results := make([]Element, 0, 100)

// 	for iter.Next() {

// 		// contents of the returned slice must not be modified, and are
// 		// only valid until the next call to Next
// 		key := iter.Key()
// 		value := iter.Value()

// 		dataKey := make([]byte, len(key)-1) // strip the prefix
// 		copy(dataKey, key[1:])              // ...

// 		dataValue := make([]byte, len(value))
// 		copy(dataValue, value)

// 		e := Element{
// 			Key:   dataKey,
// 			Value: dataValue,
// 		}
// 		results = append(results, e)
// 	}
// 	iter.Release()
// 	err := iter.Error()
// 	fault.PanicIfError("pool.FetchPrefixed", err)

// 	return results
// }

// fetch the N most recent binary key and data pairs
//
// only used by test to ensure pool contents are correct
func (p *Pool) Recent(count int) ([]Element, error) {
	if count <= 0 {
		return nil, fault.ErrInvalidCount
	}

	p.Lock()
	defer p.Unlock()

	n := 0
	a := make([]Element, 0, count)
	for element := p.lru.Front(); nil != element && n < count; element = element.Next() {

		e := Element{
			Key:   element.Value.(*Element).Key,
			Value: element.Value.(*Element).Value,
		}
		a = append(a, e)
		n += 1
	}
	return a, nil
}

// flush the pool channel
//
// create empty index and LRU cache
func (p *Pool) Flush() {
	p.Lock()
	defer p.Unlock()

	p.lru = list.List{}
	p.index = make(map[string]*list.Element)
	return
}
