// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// type for batch operations
type Batch struct {
	batch  *leveldb.Batch
	active bool
}

// begin a new batch of Adds and Removes
//
// the adds and removes will be executed only on Commit
func NewBatch() *Batch {
	return &Batch{
		batch:  &leveldb.Batch{},
		active: true,
	}
}

// add a key/value bytes pair to the database
func (batch *Batch) Add(pool *Pool, key []byte, value []byte) {
	if !batch.active {
		fault.Panic("Add to inactive batch")
	}
	batch.batch.Put(pool.prefixKey(key), value)
}

// remove a key from the database
func (batch *Batch) Remove(pool *Pool, key []byte) {
	if !batch.active {
		fault.Panic("Remove from inactive batch")
	}
	batch.batch.Delete(pool.prefixKey(key))
}

// to perform synchronous commit
var syncCommit = opt.WriteOptions{
	Sync: true,
}

// commit the accumulated Adds and removes to the database
func (batch *Batch) Commit() {
	if !batch.active {
		fault.Panic("Commit on inactive batch")
	}
	batch.active = false
	err := poolData.database.Write(batch.batch, &syncCommit)
	fault.PanicIfError("Batch.Commit", err)
}
