// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
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
	requiredConfirmations = 1
	maximumBlockRate      = 500.0 // blocks per second
)

type P2PCache struct {
	BtcDirectory string `gluamapper:"btc_directory" json:"btc_directory"`
	LtcDirectory string `gluamapper:"ltc_directory" json:"ltc_directory"`
}

// Configuration - structure for configuration file
type Configuration struct {
	AutoVerify     bool                        `gluamapper:"auto_verify" json:"auto_verify"`
	Mode           string                      `gluamapper:"mode" json:"mode"`
	P2PCache       P2PCache                    `gluamapper:"p2p_cache" json:"p2p_cache"`
	BootstrapNodes bootstrapNodesConfiguration `gluamapper:"bootstrap_nodes" json:"bootstrap_nodes"`
	Bitcoin        *currencyConfiguration      `gluamapper:"bitcoin" json:"bitcoin"`
	Litecoin       *currencyConfiguration      `gluamapper:"litecoin" json:"litecoin"`
}

type bootstrapNodesConfiguration struct {
	Bitcoin  []string `gluamapper:"bitcoin" json:"bitcoin"`
	Litecoin []string `gluamapper:"litecoin" json:"litecoin"`
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
		return fault.AlreadyInitialised
	}

	globalData.log = logger.New("payment")
	globalData.log.Info("starting…")

	if configuration.Mode == "noverify" {
		globalData.initialised = true
		return nil
	}

	// initialise the handler for each currency
	globalData.handlers = make(map[string]currencyHandler)
	if configuration.Mode != "p2p" {
		for c := currency.First; c <= currency.Last; c++ {
			switch c {
			case currency.Bitcoin:
				if configuration.Bitcoin == nil {
					return fault.MissingPaymentBitcoinSection
				}
				handler, err := newBitcoinHandler(configuration.Bitcoin)
				if err != nil {
					return err
				}
				globalData.handlers[currency.Bitcoin.String()] = handler
			case currency.Litecoin:
				if configuration.Litecoin == nil {
					return fault.MissingPaymentLitecoinSection
				}
				handler, err := newLitecoinHandler(configuration.Litecoin)
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

		btcP2pWatcher, err := newP2pWatcher(currency.Bitcoin,
			configuration.P2PCache.BtcDirectory,
			configuration.BootstrapNodes.Bitcoin)
		if err != nil {
			return err
		}
		ltcP2pWatcher, err := newP2pWatcher(currency.Litecoin,
			configuration.P2PCache.LtcDirectory,
			configuration.BootstrapNodes.Litecoin)
		if err != nil {
			return err
		}
		processes = append(processes, btcP2pWatcher, ltcP2pWatcher)
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
		return fault.NotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background if one was started
	if globalData.background != nil {
		globalData.background.StopAndWait()
	}

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
