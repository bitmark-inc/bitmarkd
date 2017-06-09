// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/logger"
	"net/http"
	"sync"
)

// globals for background proccess
type litecoinData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	// connection to litecoin daemon
	client *http.Client
	url    string

	// values from litecoind
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
var globalData litecoinData

// a block of configuration data
// this is read from a libucl configuration file
type Configuration struct {
	URL string `libucl:"url"`
}

// initialise for litecoin payments
// also calls the internal initialisePayment() and register()
func Initialise(configuration *Configuration) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("litecoin")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	globalData.url = configuration.URL
	globalData.client = &http.Client{}

	globalData.log.Debugf("url: %s", globalData.url)

	// all data initialised
	globalData.initialised = true

	globalData.log.Debug("getinfo…")

	// query litecoind for status
	// only need to have necessary fields as JSON unmarshaller will ignore excess
	var reply struct {
		Chain  string `json:"chain"`
		Blocks uint64 `json:"blocks"`
		Hash   string `json:"bestblockhash"`
	}

	err := rpc(globalData.url+"/chaininfo.json", &reply)
	if nil != err {
		return err
	}

	// check chain is ok
	if !mode.IsTesting() && "main" != reply.Chain {
		globalData.log.Errorf("Litecoin chain: %s not allowed on bitmark network", reply.Chain)
		return fault.ErrInvalidChain
	}

	globalData.log.Debugf("chain: %s  block: %d %s", reply.Chain, reply.Blocks, reply.Hash)

	// set up current top block number
	globalData.latestBlockNumber = reply.Blocks
	globalData.latestBlockHash = reply.Hash
	globalData.forward = false

	// start background processes
	globalData.log.Info("start background…")

	// list of background processes to start
	processes := background.Processes{
		&globalData,
	}

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

	// stop background
	globalData.background.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
