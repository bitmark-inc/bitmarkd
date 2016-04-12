// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool_test

import (
	"github.com/bitmark-inc/bitmarkd/pool"
	"testing"
)

// helper to add to batch
func batchAdd(batch *pool.Batch, pool *pool.Pool, key string, data string) {
	batch.Add(pool, []byte(key), []byte(data))
}

// helper to remove from batch
func batchRemove(batch *pool.Batch, pool *pool.Pool, key string) {
	batch.Remove(pool, []byte(key))
}

// test batch of writes
func TestBatch(t *testing.T) {
	setup(t)
	defer teardown(t)

	p := pool.New(pool.TestData)

	// ensure that pool was empty
	checkAgain(t, true)

	// batch
	b := pool.NewBatch()

	// add more items than batchSize
	batchAdd(b, p, "key-one", "data-one")
	batchAdd(b, p, "key-two", "data-two")
	batchAdd(b, p, "key-remove-me", "to be deleted")
	batchRemove(b, p, "key-remove-me")
	batchAdd(b, p, "key-three", "data-three")
	batchAdd(b, p, "key-one", "data-one")     // duplicate
	batchAdd(b, p, "key-three", "data-three") // duplicate
	batchAdd(b, p, "key-four", "data-four")
	batchAdd(b, p, "key-delete-this", "to be deleted")
	batchAdd(b, p, "key-five", "data-five")
	batchAdd(b, p, "key-six", "data-six")
	batchRemove(b, p, "key-delete-this")
	batchAdd(b, p, "key-seven", "data-seven")
	batchAdd(b, p, "key-one", "data-one(NEW)") // duplicate

	// ensure that pool still empty
	checkAgain(t, true)

	// write the data
	b.Commit()

	// ensure that data is correct
	checkResults(t, p)

	// recheck
	checkAgain(t, false)

	// check that restarting database keeps data
	pool.Finalise()
	pool.Initialise(databaseFileName)
	checkAgain(t, false)
}
