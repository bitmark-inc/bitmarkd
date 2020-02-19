// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockheader_test

import (
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/logger"
)

// remove all files created by test
func removeFiles() {
	os.RemoveAll("test.log")
}

// configure for testing
func setup(t *testing.T) {
	removeFiles()

	logger.Initialise(logger.Configuration{
		Directory: ".",
		File:      "test.log",
		Size:      50000,
		Count:     10,
	})

	err := blockheader.Initialise()
	if nil != err {
		t.Fatalf("initialise error: %s", err)
	}
}

// post test cleanup
func teardown(t *testing.T) {
	err := blockheader.Finalise()
	if nil != err {
		t.Fatalf("finalise error: %s", err)
	}
	logger.Finalise()
	removeFiles()
}
