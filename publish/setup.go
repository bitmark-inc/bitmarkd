// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package publish

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

// Configuration - a block of configuration data
// this is read from the configuration file
type Configuration struct {
	Broadcast  []string `gluamapper:"broadcast" json:"broadcast"`
	PrivateKey string   `gluamapper:"private_key" json:"private_key"`
	PublicKey  string   `gluamapper:"public_key" json:"public_key"`
}

// globals for background process
type publishData struct {
	sync.RWMutex // to allow locking

	log *logger.L // logger

	brdc broadcaster // for broadcasting blocks, transactions etc.

	publicKey []byte

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData publishData

// Initialise - setup peer background processes
func Initialise(configuration *Configuration, version string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	globalData.log = logger.New("publish")
	globalData.log.Info("starting…")

	if 0 == len(configuration.Broadcast) {
		globalData.log.Info("no broadcasts - disabling")
		return nil
	}

	// read the keys
	privateKey, err := zmqutil.ReadPrivateKey(configuration.PrivateKey)
	if nil != err {
		globalData.log.Errorf("read private key file: %q  error: %s", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadPublicKey(configuration.PublicKey)
	if nil != err {
		globalData.log.Errorf("read public key file: %q  error: %s", configuration.PublicKey, err)
		return err
	}
	globalData.log.Tracef("private key: %q", privateKey)
	globalData.log.Tracef("public key:  %q", publicKey)

	globalData.publicKey = publicKey

	if err := globalData.brdc.initialise(privateKey, publicKey, configuration.Broadcast); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
		&globalData.brdc,
	}

	globalData.background = background.Start(processes, globalData.log)

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

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
