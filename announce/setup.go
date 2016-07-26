// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	"net"
	"strings"
	"sync"
)

// type of listener
const (
	TypeRPC  = iota
	TypePeer = iota
)

// globals for background proccess
type announcerData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// this node's annoucements
	rpcs       []*rpcEntry
	broadcasts []*broadcastEntry
	listeners  []*listenEntry

	ann announcer

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData announcerData

// initialise the announcement system
func Initialise() error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("announcer")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	texts, err := net.LookupTXT("nodes.test.bitmark.com")
	if nil != err {
		return err
	}

	// process DNS entries
	for i, t := range texts {
		t = strings.TrimSpace(t)
		tags, err := parseTag(t)
		if nil != err {
			globalData.log.Infof("ignore TXT[%d]: %q  error: %v", i, t, err)
		} else {
			globalData.log.Infof("process TXT[%d]: %q", i, t)
			globalData.log.Infof("result[%d]: %#v", i, tags)
		}

	}

	if err := globalData.ann.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	var processes = background.Processes{
		&globalData.ann,
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// finialise - stop all background tasks
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.Stop()

	// finally...
	globalData.initialised = false

	return nil
}
