// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir_test

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
	"os"
	"testing"
	"time"
)

// test database file
const (
	databaseFileName = "test.leveldb"
)

// common test setup routines

// remove all files created by test
func removeFiles() {
	os.RemoveAll(databaseFileName)
	os.RemoveAll("test.log")
}

// configure for testing
func setup(t *testing.T, theChain ...string) {

	removeFiles()

	logger.Initialise(logger.Configuration{
		Directory: ".",
		File:      "test.log",
		Size:      50000,
		Count:     10,
	})

	if len(theChain) >= 1 {
		mode.Initialise(theChain[0])
	} else {
		mode.Initialise(chain.Bitmark)
	}

	err := storage.Initialise(databaseFileName)
	if nil != err {
		t.Fatalf("storage initialise error: %s", err)
	}

	// need to initialise block befor any tests can be performed
	err = block.Initialise()
	if nil != err {
		t.Fatalf("block initialise error: %s", err)
	}
}

// post test cleanup
func teardown(t *testing.T) {
	block.Finalise()
	storage.Finalise()
	mode.Finalise()
	logger.Finalise()
	removeFiles()

	// just to ensure background process in block has stopped
	time.Sleep(25 * time.Millisecond)
}
