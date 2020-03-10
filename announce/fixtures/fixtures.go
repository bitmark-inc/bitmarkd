// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fixtures

import (
	"fmt"
	"os"

	"github.com/bitmark-inc/logger"
)

const (
	dir         = "testing"
	logCategory = "testing"
)

func SetupTestLogger() {
	removeFiles()
	_ = os.Mkdir(dir, 0700)

	logging := logger.Configuration{
		Directory: dir,
		File:      fmt.Sprintf("%s.log", logCategory),
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

func TeardownTestLogger() {
	logger.Finalise()
	removeFiles()
}

func removeFiles() {
	err := os.RemoveAll(dir)
	if nil != err {
		fmt.Println("remove dir with error: ", err)
	}
}
