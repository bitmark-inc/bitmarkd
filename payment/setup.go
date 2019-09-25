// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

const (
	// MinimumNonceLength - least number of bytes in NONCE
	MinimumNonceLength = 8
	// MaximumNonceLength - most number of bytes in NONCE
	MaximumNonceLength = 64
)
const (
	requiredConfirmations = 3
	maximumBlockRate      = 500.0 // blocks per second
)

// Configuration - structure for configuration file
type Configuration struct {
	Mode      string                  `gluamapper:"mode" hcl:"mode" json:"mode"`
	Discovery *discoveryConfiguration `gluamapper:"discovery" hcl:"discovery" json:"discovery"`
	Bitcoin   *currencyConfiguration  `gluamapper:"bitcoin" hcl:"bitcoin" json:"bitcoin"`
	Litecoin  *currencyConfiguration  `gluamapper:"litecoin" hcl:"litecoin" json:"litecoin"`
}

type discoveryConfiguration struct {
	ReqEndpoint string `gluamapper:"req_endpoint" hcl:"req_endpoint" json:"req_endpoint"`
	SubEndpoint string `gluamapper:"sub_endpoint" hcl:"sub_endpoint" json:"sub_endpoint"`
}

type currencyConfiguration struct {
	URL string `gluamapper:"url" json:"url"`
}

type globalDataType struct {
	sync.RWMutex // to allow locking

	log        *logger.L
	handlers   map[string]currencyHandler
	background *background.T

	// set once during initialise
	initialised bool
}

var globalData globalDataType

// Initialise - setup the payment system
func Initialise(configuration *Configuration) error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("payment")
	globalData.log.Info("starting…")

	// initialise the handler for each currency
	globalData.handlers = make(map[string]currencyHandler)
	if configuration.Mode != "p2p" {
		useDiscovery := configuration.Mode == "discovery"
		for c := currency.First; c <= currency.Last; c++ {
			switch c {
			case currency.Bitcoin:
				handler, err := newBitcoinHandler(useDiscovery, configuration.Bitcoin)
				if err != nil {
					return err
				}
				globalData.handlers[currency.Bitcoin.String()] = handler
			case currency.Litecoin:
				handler, err := newLitecoinHandler(useDiscovery, configuration.Litecoin)
				if err != nil {
					return err
				}
				globalData.handlers[currency.Litecoin.String()] = handler
			default: // only fails if new module not correctly installed
				logger.Panicf("missing payment initialiser for Currency: %s", c.String())
			}
		}
	}

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{}

	switch configuration.Mode {
	case "p2p":
		globalData.log.Info("p2p watcher…")

		btcP2pWatcher, err := newP2pWatcher(currency.Bitcoin)
		if err != nil {
			return err
		}
		ltcP2pWatcher, err := newP2pWatcher(currency.Litecoin)
		if err != nil {
			return err
		}
		processes = append(processes, btcP2pWatcher, ltcP2pWatcher)
	case "discovery":
		globalData.log.Info("discovery…")
		discoverer, err := newDiscoverer(configuration.Discovery.SubEndpoint, configuration.Discovery.ReqEndpoint)
		if err != nil {
			return err
		}
		processes = append(processes, discoverer)
	case "rest":
		globalData.log.Info("checker…")
		processes = append(processes, &checker{})
	default:
		logger.Panicf("unsupported payment verification mode: %s", configuration.Mode)
	}

	// all data initialised
	globalData.initialised = true

	// start background
	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// Finalise - stop all background tasks
func Finalise() error {
	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.background.StopAndWait()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
