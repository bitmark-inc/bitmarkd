// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer_test

import (
	"os"
	"testing"

	"github.com/bitmark-inc/logger"
)

const (
	dir      = "testing"
	category = "testing"
)

func setupTestLogger() {
	removeFiles()
	_ = os.Mkdir(dir, 0700)

	logging := logger.Configuration{
		Directory: dir,
		File:      "testing.log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)
}

func teardownTestLogger() {
	removeFiles()
}

func removeFiles() {
	_ = os.RemoveAll(dir)
}

func TestMain(m *testing.M) {
	setupTestLogger()
	rc := m.Run()
	teardownTestLogger()
	os.Exit(rc)
}
