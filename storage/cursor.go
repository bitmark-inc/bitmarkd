// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"math/big"

	"github.com/syndtr/goleveldb/leveldb/util"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// cursor structure
type FetchCursor struct {
	pool     *PoolHandle
	maxRange util.Range
}

// initialise a cursor to the start of a key range
func (p *PoolHandle) NewFetchCursor() *FetchCursor {

	return &FetchCursor{
		pool: p,
		maxRange: util.Range{
			Start: []byte{p.prefix}, // Start of key range, included in the range
			Limit: p.limit,          // Limit of key range, excluded from the range
		},
	}
}

// initialise a cursor to the start of a key range
func (p *PoolNB) NewFetchCursor() *FetchCursor {
	return p.pool.NewFetchCursor()
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

	if nil == cursor.pool.database {
		return nil, nil
	}

	iter := cursor.pool.database.NewIterator(&cursor.maxRange, nil)

	results := make([]Element, 0, count)
	n := 0
iterating:
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
			break iterating
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

// map a function over all elements in the range
func (cursor *FetchCursor) Map(f func(key []byte, value []byte) error) error {
	if nil == cursor {
		return fault.ErrInvalidCursor
	}

	if nil == cursor.pool.database {
		return nil
	}

	iter := cursor.pool.database.NewIterator(&cursor.maxRange, nil)

	var err error
iterating:
	for iter.Next() {

		// contents of the returned slice must not be modified, and are
		// only valid until the next call to Next
		key := iter.Key()
		value := iter.Value()

		dataKey := make([]byte, len(key)-1) // strip the prefix
		copy(dataKey, key[1:])              // ...

		dataValue := make([]byte, len(value))
		copy(dataValue, value)

		err = f(dataKey, dataValue)
		if nil != err {
			break iterating
		}
	}
	iter.Release()
	if nil == err {
		err = iter.Error()
	}
	return err
}
