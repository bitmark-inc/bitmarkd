// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main_test

import (
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/logger"
)

// remove all files created by test
func removeFiles() {
	os.RemoveAll("test.log")
}

// configure for testing
func setup(t *testing.T, testnet bool) {
	removeFiles()

	logger.Initialise(logger.Configuration{
		Directory: ".",
		File:      "test.log",
		Size:      50000,
		Count:     10,
	})

	if testnet {
		mode.Initialise(chain.Local)
	} else {
		mode.Initialise(chain.Bitmark)
	}
}

// post test cleanup
func teardown(t *testing.T) {
	mode.Finalise()
	logger.Finalise()
	removeFiles()
}
