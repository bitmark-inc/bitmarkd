// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"path"
	"sync"

	"github.com/bitmark-inc/bitmarkd/messagebus"

	"github.com/bitmark-inc/bitmarkd/announce/domain"

	"github.com/bitmark-inc/bitmarkd/announce/broadcast"
	"github.com/bitmark-inc/bitmarkd/announce/parameter"

	"github.com/bitmark-inc/bitmarkd/announce/rpc"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

const (
	logCategory = "announce"
)

// file for storing saves peers
const backupFile = "peers.json"

// globals for background process
type announcerData struct {
	sync.RWMutex // to allow locking

	log *logger.L

	// RPC interface
	rpcs rpc.RPC

	// Receptor interface
	receptors receptor.Receptor

	backupFile string

	// data for thread
	brdc background.Process

	domain background.Process

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData announcerData

// Initialise - set up the announcement system
// pass a fully qualified domain for root node list
// or empty string for no root nodes
func Initialise(domainName, cacheDirectory string, f func(string) ([]string, error)) error {
	globalData.Lock()
	defer globalData.Unlock()

	var err error

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	globalData.log = logger.New(logCategory)
	globalData.log.Info("starting…")

	globalData.receptors = receptor.New(globalData.log)
	globalData.backupFile = path.Join(cacheDirectory, backupFile)

	globalData.log.Info("start restoring backup data…")
	if err := receptor.Restore(globalData.backupFile, globalData.receptors); err != nil {
		globalData.log.Errorf("fail to restore backup data: %s", err.Error())
	}

	globalData.rpcs = rpc.New()

	globalData.domain, err = domain.New(
		globalData.log,
		domainName,
		globalData.receptors,
		f,
	)
	if nil != err {
		return err
	}

	globalData.brdc = broadcast.New(
		globalData.log,
		globalData.receptors,
		globalData.rpcs,
		parameter.InitialiseInterval,
		parameter.PollingInterval,
	)

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
		globalData.domain, globalData.brdc,
	}

	globalData.background = background.Start(processes, messagebus.Bus.Announce.Chan())

	return nil
}

// Finalise - stop all background tasks
func Finalise() error {
	if !globalData.initialised {
		return fault.NotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()

	// release message bus
	messagebus.Bus.Announce.Release()

	globalData.log.Info("start backing up peer data…")
	if err := receptor.Backup(globalData.backupFile, globalData.receptors.Connectable()); err != nil {
		globalData.log.Errorf("fail to backup peer data: %s", err.Error())
	}

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
