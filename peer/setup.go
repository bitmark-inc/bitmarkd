// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	"sync"
)

// hardwired connections
// public key in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32//
// this is read from a libucl configuration file
type Connection struct {
	PublicKey string `libucl:"public_key"`
	Address   string `libucl:"address"`
}

// for announcements
type Announce struct {
	Broadcast []string `libucl:"broadcast"`
	Listen    []string `libucl:"listen"`
}

// server identification in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
// a block of configuration data
// this is read from a libucl configuration file
type Configuration struct {
	//MaximumConnections int          `libucl:"maximum_connections"`
	Broadcast  []string     `libucl:"broadcast"`
	Listen     []string     `libucl:"listen"`
	Announce   Announce     `libucl:"announce"`
	PrivateKey string       `libucl:"private_key"`
	PublicKey  string       `libucl:"public_key"`
	Subscribe  []Connection `libucl:"subscribe"`
	Connect    []Connection `libucl:"connect"`
}

// globals for background proccess
type proofData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	brd    broadcaster // for broadcasting blocks, transactions etc.
	listen listener    // for RPC responses
	conn   connector   // for RPC requests
	subs   subscriber  // for subscriptions

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData proofData

// initialise proofer backgrouds processes
func Initialise(configuration *Configuration) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("peer")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	if err := globalData.brd.initialise(configuration); nil != err {
		return err
	}
	if err := globalData.listen.initialise(configuration); nil != err {
		return err
	}
	if err := globalData.conn.initialise(configuration); nil != err {
		return err
	}
	if err := globalData.subs.initialise(configuration); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	var processes = background.Processes{
		&globalData.brd,
		&globalData.listen,
		&globalData.conn,
		&globalData.subs,
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
