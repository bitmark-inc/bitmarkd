// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"bytes"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
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
//
// note all must be exported (i.e. initial capital) or initialisation will panic
type pools struct {
	Blocks            *PoolHandle `prefix:"B"`
	BlockOwnerAccount *PoolHandle `prefix:"G"`
	BlockOwnerPayment *PoolHandle `prefix:"H"`
	BlockOwnerTxIndex *PoolHandle `prefix:"I"`
	Assets            *PoolHandle `prefix:"A"`
	Transactions      *PoolHandle `prefix:"T"`
	OwnerCount        *PoolHandle `prefix:"N"`
	Ownership         *PoolHandle `prefix:"K"`
	OwnerDigest       *PoolHandle `prefix:"D"`
	TestData          *PoolHandle `prefix:"Z"`
}

// the instance
var Pool pools

// for database version
var versionKey = []byte{0x00, 'V', 'E', 'R', 'S', 'I', 'O', 'N'}
var currentVersion = []byte{0x00, 0x00, 0x00, 0x03}

// holds the database handle
var poolData struct {
	sync.RWMutex
	database *leveldb.DB
}

// open up the database connection
//
// this must be called before any pool.New() is created
func Initialise(database string) error {
	poolData.Lock()
	defer poolData.Unlock()

	if nil != poolData.database {
		return fault.ErrAlreadyInitialised
	}

	db, err := leveldb.RecoverFile(database, nil)
	// db, err := leveldb.OpenFile(database, nil)
	if nil != err {
		return err
	}

	poolData.database = db

	// ensure that the database is compatible
	versionValue, err := poolData.database.Get(versionKey, nil)
	if leveldb.ErrNotFound == err {
		err = poolData.database.Put(versionKey, currentVersion, nil)
		if nil != err {
			return err
		}
	} else if nil != err {
		return err
	} else if !bytes.Equal(versionValue, currentVersion) {
		return fmt.Errorf("incompatible database version: expected: 0x%x  actual: 0x%x", currentVersion, versionValue)
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
			return fmt.Errorf("pool: %v  has invalid prefix: %q", fieldInfo, prefixTag)
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

	return nil
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
	poolData.RLock()
	defer poolData.RUnlock()
	if nil == poolData.database {
		return
	}
	err := poolData.database.Put(p.prefixKey(key), value, nil)
	logger.PanicIfError("pool.Put", err)
}

// remove a key from the database
func (p *PoolHandle) Delete(key []byte) {
	poolData.RLock()
	defer poolData.RUnlock()
	err := poolData.database.Delete(p.prefixKey(key), nil)
	logger.PanicIfError("pool.Delete", err)
}

// read a value for a given key
//
// this returns the actual element - copy the result if it must be preserved
func (p *PoolHandle) Get(key []byte) []byte {
	poolData.RLock()
	defer poolData.RUnlock()
	if nil == poolData.database {
		return nil
	}
	value, err := poolData.database.Get(p.prefixKey(key), nil)
	if leveldb.ErrNotFound == err {
		return nil
	}
	logger.PanicIfError("pool.Get", err)
	return value
}

// Check if a key exists
func (p *PoolHandle) Has(key []byte) bool {
	poolData.RLock()
	defer poolData.RUnlock()
	if nil == poolData.database {
		return false
	}
	value, err := poolData.database.Has(p.prefixKey(key), nil)
	logger.PanicIfError("pool.Has", err)
	return value
}

// get the last element in a pool
func (p *PoolHandle) LastElement() (Element, bool) {
	maxRange := util.Range{
		Start: []byte{p.prefix}, // Start of key range, included in the range
		Limit: p.limit,          // Limit of key range, excluded from the range
	}

	poolData.RLock()
	defer poolData.RUnlock()
	if nil == poolData.database {
		return Element{}, false
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
	logger.PanicIfError("pool.LastElement", err)
	return result, found
}
