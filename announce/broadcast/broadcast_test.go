// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package broadcast_test

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/bitmark-inc/bitmarkd/background"

	"github.com/bitmark-inc/bitmarkd/messagebus"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"

	"github.com/bitmark-inc/bitmarkd/announce/rpc"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/announce/broadcast"
)

const (
	dir         = "testing"
	logCategory = "testing"
)

func setupTestLogger() {
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

func teardownTestLogger() {
	logger.Finalise()
	removeFiles()
}

func removeFiles() {
	err := os.RemoveAll(dir)
	if nil != err {
		fmt.Println("remove dir with error: ", err)
	}
}

func TestRunWhenSendingShutdown(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	log := logger.New(logCategory)
	b := broadcast.New(
		log,
		receptor.New(log),
		rpc.New(),
		fingerprint.Type{1, 2, 3, 4},
		broadcast.UsePeers,
	)

	ch := make(chan messagebus.Message)
	shutdown := make(chan struct{})
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, b background.Process, wg *sync.WaitGroup, sh <-chan struct{}) {
		b.Run(ch, sh)
		wg.Done()
	}(ch, b, wg, shutdown)

	shutdown <- struct{}{}
	wg.Wait()
}
