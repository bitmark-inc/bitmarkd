// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/syndtr/goleveldb/leveldb/util"
	"math/big"
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
