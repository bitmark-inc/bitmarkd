// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	ldb_util "github.com/syndtr/goleveldb/leveldb/util"
)

// for Database
type Access interface {
	Abort()
	Begin() error
	Commit() error
	Delete([]byte)
	DumpTx() []byte
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
	InUse() bool
	Iterator(*ldb_util.Range) iterator.Iterator
	Put([]byte, []byte)
}

type AccessData struct {
	sync.Mutex
	inUse bool
	db    *leveldb.DB
	batch *leveldb.Batch
	cache Cache
}

func newDA(db *leveldb.DB, trx *leveldb.Batch, cache Cache) Access {
	return &AccessData{
		inUse: false,
		db:    db,
		batch: trx,
		cache: cache,
	}
}

func (d *AccessData) Begin() error {
	d.Lock()
	defer d.Unlock()

	if d.inUse {
		return fmt.Errorf("batch already in use")
	}

	d.inUse = true
	return nil
}

func (d *AccessData) Put(key []byte, value []byte) {
	d.cache.Set(dbPut, string(key), value)
	d.batch.Put(key, value)
}

func (d *AccessData) Delete(key []byte) {
	d.cache.Set(dbDelete, string(key), []byte{})
	d.batch.Delete(key)
}

func (d *AccessData) Commit() error {
	return d.db.Write(d.batch, nil)
}

func (d *AccessData) DumpTx() []byte {
	return d.batch.Dump()
}

func (d *AccessData) Get(key []byte) ([]byte, error) {
	val, found := d.getFromCache(key)
	if found {
		return val, nil
	}
	return d.getFromDB(key)
}

func (d *AccessData) getFromCache(key []byte) ([]byte, bool) {
	return d.cache.Get(string(key))
}

func (d *AccessData) getFromDB(key []byte) ([]byte, error) {
	return d.db.Get(key, nil)
}

func (d *AccessData) Iterator(searchRange *ldb_util.Range) iterator.Iterator {
	return d.db.NewIterator(searchRange, nil)
}

func (d *AccessData) Has(key []byte) (bool, error) {
	_, found := d.getFromCache(key)
	if found {
		return true, nil
	}
	return d.db.Has(key, nil)
}

func (d *AccessData) InUse() bool {
	return d.inUse
}

func (d *AccessData) Abort() {
	d.Lock()
	defer d.Unlock()

	d.batch.Reset()
	d.cache.Clear()
	d.inUse = false
}
