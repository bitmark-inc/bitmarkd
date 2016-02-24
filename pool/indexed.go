// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"sync"
)

// prefix bute is any value to separate different databases
// code in the second key byte
//   data:       data-key  -> data
//   timestamp:  timestamp -> data-key
//   index:      index-key -> timestamp
const (
	prefixSize         = 2 // [...]byte{prefix byte, prefix code}
	dataCode           = byte('D')
	indexCode          = byte('K')
	indexLimitCode     = byte(indexCode + 1)
	timestampCode      = byte('T')
	timestampLimitCode = byte(timestampCode + 1)
)

// the pool handle
type IndexedPool struct {
	sync.RWMutex
	prefix byte
}

// create a new indexed pool with a specific key prefix
//
// Separate indexes are created to provide a timestamp ordering
func NewIndexed(prefix nameb) *IndexedPool {

	poolData.Lock()
	defer poolData.Unlock()
	if nil == poolData.database {
		fault.Panic("pool.New - not initialised")
	}

	pool := IndexedPool{
		prefix: byte(prefix),
	}

	return &pool
}

// add a key/value bytes pair to the database
func (p *IndexedPool) Add(key []byte, value []byte) (bool, error) {
	p.Lock()
	defer p.Unlock()

	newAddition := false

	// compute keys
	prefixedData, err := p.dataKey(key)
	if nil != err {
		return newAddition, err
	}

	prefixedIndex, err := p.indexKey(key)
	if nil != err {
		return newAddition, err
	}

	prefixedTimestamp := p.timestampKey(gnomon.NewCursor())

	// batch to update database
	batch := leveldb.Batch{}

	// get old timestamp
	oldTimestamp, err := poolData.database.Get(prefixedIndex, nil)
	if nil == err {
		batch.Delete(oldTimestamp)
	} else {
		newAddition = true
	}

	batch.Put(prefixedIndex, prefixedTimestamp)
	batch.Put(prefixedTimestamp, prefixedData)
	batch.Put(prefixedData, value)

	return newAddition, poolData.database.Write(&batch, nil)
}

// remove a key from the database
func (p *IndexedPool) Remove(key []byte) error {
	p.Lock()
	defer p.Unlock()

	// delete from database
	prefixedData, err := p.dataKey(key)
	if nil != err {
		return err
	}

	prefixedIndex, err := p.indexKey(key)
	if nil != err {
		return err
	}

	// batch to update database
	batch := leveldb.Batch{}

	// get old timestamp
	oldTimestamp, err := poolData.database.Get(prefixedIndex, nil)
	if nil == err {
		batch.Delete(oldTimestamp)
	}

	batch.Delete(prefixedData)
	batch.Delete(prefixedIndex)

	return poolData.database.Write(&batch, nil)
}

// read a value for a given key
func (p *IndexedPool) Get(key []byte) ([]byte, error) {
	p.Lock()
	defer p.Unlock()

	prefixedData, err := p.dataKey(key)
	if nil != err {
		return nil, err
	}

	value, err := poolData.database.Get(prefixedData, nil)
	if leveldb.ErrNotFound == err {
		return nil, fault.ErrKeyNotFound
	}
	return value, err
}

// fetch the N most recent string key and data pairs
func (p *IndexedPool) Recent(start *gnomon.Cursor, count int, convert func(key []byte, value []byte) interface{}) ([]interface{}, *gnomon.Cursor, error) {
	if nil == start {
		start = &gnomon.Cursor{}
	}

	if count <= 0 {
		return nil, nil, fault.ErrInvalidCount
	}

	p.RLock()
	defer p.RUnlock()

	timestampRange := util.Range{
		Start: p.timestampKey(start),
		Limit: []byte{p.prefix, timestampLimitCode},
	}

	iter := poolData.database.NewIterator(&timestampRange, nil)
	defer iter.Release()

	n := 0
	a := make([]interface{}, 0, count)

	thisKey := make([]byte, prefixSize+8+4) // prefix + int64 + int32

	for iter.Next() {
		prefixedData := iter.Value()
		dataValue, err := poolData.database.Get(prefixedData, nil)
		if leveldb.ErrNotFound == err {
			continue
		}

		thisKey = iter.Key()

		if nil == convert {
			dataKey := make([]byte, len(prefixedData)-prefixSize)
			copy(dataKey, prefixedData[prefixSize:])
			e := Element{
				Key:   dataKey,
				Value: dataValue,
			}
			a = append(a, e)
		} else {
			a = append(a, convert(prefixedData[prefixSize:], dataValue))
		}
		n += 1
		if n >= count {
			break
		}

	}
	iter.Release()

	// only increment the cursor if actual data was found
	// otherwise preserve
	if n > 0 {
		nextStart := gnomon.Cursor{}
		err := nextStart.UnmarshalBinary(thisKey[prefixSize:])
		if nil != err {
			return nil, nil, err
		}

		nextStart.Next()

		return a, &nextStart, iter.Error()
	}
	return a, start, iter.Error()

}

// flush any unsaved data
func (p *IndexedPool) Flush() error {
	// p.Lock()
	// defer p.Unlock()
	return nil
}

// create a timestamp key
func (p *IndexedPool) timestampKey(cursor *gnomon.Cursor) []byte {

	buffer, err := cursor.MarshalBinary()
	// error should always be nil, but just in case the lower level
	// code changes -> panic
	if nil != err {
		panic(err)
	}

	// big endian encode cursor
	return append([]byte{p.prefix, timestampCode}, buffer...)
}

// create a data key
func (p *IndexedPool) dataKey(key []byte) ([]byte, error) {

	prefixedKey := make([]byte, prefixSize, len(key)+prefixSize)
	prefixedKey[0] = p.prefix
	prefixedKey[1] = dataCode

	return append(prefixedKey, key...), nil
}

// create an index key
func (p *IndexedPool) indexKey(key []byte) ([]byte, error) {

	prefixedKey := make([]byte, prefixSize, len(key)+prefixSize)
	prefixedKey[0] = p.prefix
	prefixedKey[1] = indexCode

	return append(prefixedKey, key...), nil
}
