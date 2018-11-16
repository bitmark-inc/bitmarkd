// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package proof

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// server identification in Z85 (ZeroMQ Base-85 Encoding) see: http://rfc.zeromq.org/spec:32
// a block of configuration data
// this is read from the configuration file
type Configuration struct {
	Publish     []string          `gluamapper:"publish" json:"publish"`
	Submit      []string          `gluamapper:"submit" json:"submit"`
	PrivateKey  string            `gluamapper:"private_key" json:"private_key"`
	PublicKey   string            `gluamapper:"public_key" json:"public_key"`
	SigningKey  string            `gluamapper:"signing_key" json:"signing_key"`
	PaymentAddr map[string]string `gluamapper:"payment_address" json:"payment_address"`
}

// globals for background proccess
type proofData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// for publisher
	pub publisher

	// for submission
	sub submission

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

	globalData.log = logger.New("proof")
	globalData.log.Info("starting…")

	if err := globalData.pub.initialise(configuration); nil != err {
		return err
	}
	if err := globalData.sub.initialise(configuration); nil != err {
		return err
	}

	// create the job queue
	initialiseJobQueue()

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	processes := background.Processes{
		&globalData.pub,
		&globalData.sub,
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

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
