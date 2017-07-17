// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

const (
	NonceLength           = 64 // hex bytes
	requiredConfirmations = 3
	maximumBlockRate      = 500.0 // blocks per second
)

type Configuration struct {
	UseDiscovery bool                    `libucl:"use_discovery" hcl:"use_discovery" json:"use_discovery"`
	Discovery    *discoveryConfiguration `libucl:"discovery" hcl:"discovery" json:"discovery"`
	Bitcoin      *currencyConfiguration  `libucl:"bitcoin" hcl:"bitcoin" json:"bitcoin"`
	Litecoin     *currencyConfiguration  `libucl:"litecoin" hcl:"litecoin" json:"litecoin"`
}

type discoveryConfiguration struct {
	ReqEndpoint string `libucl:"req_endpoint" hcl:"req_endpoint" json:"req_endpoint"`
	SubEndpoint string `libucl:"sub_endpoint" hcl:"sub_endpoint" json:"sub_endpoint"`
}

type currencyConfiguration struct {
	URL string `libucl:"url" json:"url"`
}

type globalDataType struct {
	log        *logger.L
	handlers   map[string]currencyHandler
	background *background.T
}

var globalData globalDataType

func Initialise(configuration *Configuration) error {
	globalData.log = logger.New("payment")
	if globalData.log == nil {
		return fault.ErrInvalidLoggerChannel
	}

	// initialise the handler for each currency
	globalData.handlers = make(map[string]currencyHandler)
	for c := currency.First; c <= currency.Last; c++ {
		switch c {
		case currency.Bitcoin:
			handler, err := newBitcoinHandler(configuration.UseDiscovery, configuration.Bitcoin)
			if err != nil {
				return err
			}
			globalData.handlers[currency.Bitcoin.String()] = handler
		case currency.Litecoin:
			handler, err := newLitecoinHandler(configuration.UseDiscovery, configuration.Litecoin)
			if err != nil {
				return err
			}
			globalData.handlers[currency.Litecoin.String()] = handler
		default: // only fails if new module not correctly installed
			logger.Panicf("missing payment initialiser for Currency: %s", c.String())
		}
	}

	// set up background processes
	processes := background.Processes{}
	if configuration.UseDiscovery {
		discoverer, err := newDiscoverer(globalData.log, configuration.Discovery.SubEndpoint, configuration.Discovery.ReqEndpoint)
		if err != nil {
			return err
		}
		processes = append(processes, discoverer)
	} else {
		processes = append(processes, &checker{})
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

func Finalise() {
	globalData.background.Stop()

	globalData.log.Flush()
}
