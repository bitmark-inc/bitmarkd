// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"net/http"
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// global constants
const (
	bitcoinMinimumVersion = 120100 // do not start if bitcoind olde
)

// globals for background proccess
type bitcoinData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// connection to bitcoin daemon
	client *http.Client
	url    string

	// values from bitcoind
	latestBlockNumber uint64
	latestBlockHash   string

	// scanning direction
	forward bool

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData bitcoinData

// a block of configuration data
// this is read from a libucl configuration file
type Configuration struct {
	URL string `libucl:"url" hcl:"url" json:"url"`
}

// initialise for bitcoin payments
// also calls the internal initialisePayment() and register()
func Initialise(configuration *Configuration) error {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("bitcoin")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.url = configuration.URL
	globalData.client = &http.Client{}
	globalData.initialised = true

	var chain chainInfo
	if err := util.FetchJSON(globalData.client, globalData.url+"/chaininfo.json", &chain); err != nil {
		return err
	}
	globalData.log.Debugf("chain info: %+v", chain)

	// TODO: how to get bitoind version?
	// // check version is sufficient
	// if chain.Version < bitcoinMinimumVersion {
	// 	globalData.log.Errorf("Bitcoin version: %d < allowed: %d", chain.Version, bitcoinMinimumVersion)
	// 	return fault.ErrInvalidVersion
	// }
	// globalData.log.Infof("Bitcoin version: %d", chain.Version)
	// globalData.log.Infof("Bitcoin block height: %d", chain.Blocks)

	// set up current block number
	globalData.latestBlockNumber = chain.Blocks
	globalData.latestBlockHash = chain.Hash
	globalData.forward = false

	// set up background processes
	processes := background.Processes{}
	processes = append(processes, &globalData)
	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// finalise - stop all background tasks
// also calls the internal finalisePayment()
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	globalData.background.Stop()

	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
