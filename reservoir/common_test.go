// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir_test

import (
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/reservoir/mocks"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
)

// test database file
const (
	testingDirName   = "testing"
	databaseFileName = testingDirName + "/test"
)

// common test setup routines

// remove all files created by test
func removeFiles() {
	os.RemoveAll(testingDirName)
}

// configure for testing
func setup(t *testing.T, theChain ...string) {

	removeFiles()
	os.Mkdir(testingDirName, 0o700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      "testing.log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}
	// start logging
	if err := logger.Initialise(logging); err != nil {
		panic("logger setup failed: " + err.Error())
	}

	if len(theChain) >= 1 {
		mode.Initialise(theChain[0])
	} else {
		mode.Initialise(chain.Bitmark)
	}

	// open database
	err := storage.Initialise(databaseFileName, false)
	if err != nil {
		t.Fatalf("storage initialise error: %s", err)
	}

	// need to initialise block before any tests can be performed
	err = blockheader.Initialise()
	if err != nil {
		t.Fatalf("blockheader initialise error: %s", err)
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	handle := mocks.NewMockHandle(ctl)
	handle.EXPECT().LastElement().Return(storage.Element{}, false).Times(1)
	err = block.Initialise(handle)
	if err != nil {
		t.Fatalf("block initialise error: %s", err)
	}
}

// post test cleanup
func teardown() {
	block.Finalise()
	blockheader.Finalise()
	storage.Finalise()
	mode.Finalise()
	logger.Finalise()
	removeFiles()

	// just to ensure background process in block has stopped
	time.Sleep(25 * time.Millisecond)
}
