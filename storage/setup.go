// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"reflect"
	"sync"
)

// a binary data item
type Element struct {
	Key   []byte
	Value []byte
}

// a pool handle
type PoolHandle struct {
	prefix byte
	limit  []byte
}

// exported storage pools
type pools struct {
	Blocks               *PoolHandle `prefix:"B"`
	VerifiedTransactions *PoolHandle `prefix:"V"`
	Peer                 *PoolHandle `prefix:"P"`
	TestData             *PoolHandle `prefix:"Z"`
}

// the instance
var Pool pools

// for database version
var versionKey = []byte{0x00, 'V', 'E', 'R', 'S', 'I', 'O', 'N'}
var currentVersion = []byte{0x00, 0x00, 0x00, 0x02}

// holds the database handle
var poolData struct {
	sync.Mutex
	database *leveldb.DB
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

	// this will be a struct type
	poolType := reflect.TypeOf(Pool)

	// get write acces by using pointer + Elem()
	poolValue := reflect.ValueOf(&Pool).Elem()

	// scan each field
	for i := 0; i < poolType.NumField(); i += 1 {

		fieldInfo := poolType.Field(i)

		prefixTag := fieldInfo.Tag.Get("prefix")
		if 1 != len(prefixTag) {
			fault.Panicf("pool: %v	has invalid prefix: %q", fieldInfo, prefixTag)
		}

		prefix := prefixTag[0]
		limit := []byte(nil)
		if prefix < 255 {
			limit = []byte{prefix + 1}
		}

		p := &PoolHandle{
			prefix: prefix,
			limit:  limit,
		}
		newPool := reflect.ValueOf(p)

		poolValue.Field(i).Set(newPool)
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

// prepend the prefix onto the key
func (p *PoolHandle) prefixKey(key []byte) []byte {
	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	return append(prefixedKey, key...)
}

// store a key/value bytes pair to the database
func (p *PoolHandle) Put(key []byte, value []byte) {
	err := poolData.database.Put(p.prefixKey(key), value, nil)
	fault.PanicIfError("pool.Add", err)
}

// remove a key from the database
func (p *PoolHandle) Delete(key []byte) {
	err := poolData.database.Delete(p.prefixKey(key), nil)
	fault.PanicIfError("pool.Remove", err)
}

// read a value for a given key
//
// this returns the actual element - copy the result if it must be preserved
func (p *PoolHandle) Get(key []byte) []byte {
	value, err := poolData.database.Get(p.prefixKey(key), nil)
	if leveldb.ErrNotFound == err {
		return nil
	}
	fault.PanicIfError("pool.Get", err)
	return value
}

// Check if a key exists
func (p *PoolHandle) Has(key []byte) bool {
	value, err := poolData.database.Has(p.prefixKey(key), nil)
	fault.PanicIfError("pool.Has", err)
	return value
}

// get the last element in a pool
func (p *PoolHandle) LastElement() (Element, bool) {
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
