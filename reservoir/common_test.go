// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir_test

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/storage"
	"os"
	"testing"
)

// test database file
const (
	databaseFileName = "test.leveldb"
)

// common test setup routines

// remove all files created by test
func removeFiles() {
	os.RemoveAll(databaseFileName)
}

// configure for testing
func setup(t *testing.T) {
	removeFiles()
	err := storage.Initialise(databaseFileName)
	if nil != err {
		t.Fatalf("storage initialise error: %s", err)
	}

	// need to initialise block befor any tests can be performed
	err = block.Initialise()
	if nil != err {
		t.Fatalf("block initialise error: %v", err)
	}
}

// post test cleanup
func teardown(t *testing.T) {
	block.Finalise()
	storage.Finalise()
	removeFiles()
}
