// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"fmt"
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
)

// test database file
const (
	testingDirName   = "testing"
	databaseFileName = testingDirName + "/test"
)

// Test main entrypoint
func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		os.Exit(1)
	}
	result := m.Run()
	teardown()
	os.Exit(result)
}

// remove all files created by test
func removeFiles() {
	os.RemoveAll(testingDirName)
}

// configure for testing
func setup() error {
	removeFiles()
	os.Mkdir(testingDirName, 0o700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      "testing.log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "trace",
		},
	}

	// start logging
	_ = logger.Initialise(logging)

	_ = mode.Initialise("testing")

	// open database
	err := storage.Initialise(databaseFileName, false)
	if err != nil {
		return fmt.Errorf("storage initialise error: %s", err.Error())
	}

	return nil
}

// post test cleanup
func teardown() {
	Finalise()
	logger.Finalise()
	removeFiles()
}
