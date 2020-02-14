// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce_test

import (
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/announce/broadcast"
	"github.com/stretchr/testify/assert"

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

func TestInitialise(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	err := announce.Initialise("domain.not.exist", "cache", broadcast.UsePeers, f)
	assert.Nil(t, err, "wrong Initialise")
}

func TestInitialiseWhenSecondTime(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	f := func(_ string) ([]string, error) { return []string{}, nil }

	_ = announce.Initialise("domain.not.exist", "cache", broadcast.UsePeers, f)

	err := announce.Initialise("domain.not.exist", "cache", broadcast.UsePeers, f)
	assert.Equal(t, fault.AlreadyInitialised, err, "wrong second Initialise")
}

func TestFinalise(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	err := announce.Finalise()
	assert.Nil(t, err, "wrong Finalise")
}
