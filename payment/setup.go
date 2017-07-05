// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/payment/bitcoin"
	"github.com/bitmark-inc/bitmarkd/payment/discovery"
	"github.com/bitmark-inc/bitmarkd/payment/litecoin"
	"github.com/bitmark-inc/logger"
)

// maximum values
const (
	ReceiptLength         = 64 // hex bytes
	NonceLength           = 64 // hex bytes
	RequiredConfirmations = 3
)

// configuration for each sub-module
type Configuration struct {
	UseDiscovery bool                    `libucl:"use_discovery" hcl:"use_discovery" json:"use_discovery"`
	Discovery    *DiscoveryConfiguration `libucl:"discovery" hcl:"discovery" json:"discovery"`
	Bitcoin      *bitcoin.Configuration  `libucl:"bitcoin" hcl:"bitcoin" json:"bitcoin"`
	Litecoin     *litecoin.Configuration `libucl:"litecoin" hcl:"litecoin" json:"litecoin"`
}

type DiscoveryConfiguration struct {
	ReqEndpoint string `libucl:"req_endpoint" hcl:"req_endpoint" json:"req_endpoint"`
	SubEndpoint string `libucl:"sub_endpoint" hcl:"sub_endpoint" json:"sub_endpoint"`
}

// globals
type globalDataType struct {
	log          *logger.L
	background   *background.T
	useDiscovery bool
}

// gobal storage
var globalData globalDataType

// create the tables
func Initialise(configuration *Configuration) error {
	globalData.log = logger.New("payment")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	if configuration.UseDiscovery { // Connect to discovery proxy
		discoverer, err := discovery.NewDiscoverer(configuration.Discovery.ReqEndpoint, configuration.Discovery.SubEndpoint)
		if err != nil {
			globalData.log.Info(err.Error())
			return err
		}

		processes := background.Processes{}
		processes = append(processes, discoverer)
		globalData.background = background.Start(processes, discoverer)
	} else { // Connect to real blockchains
		for c := currency.First; c <= currency.Last; c++ {
			switch c {
			case currency.Bitcoin:
				err := bitcoin.Initialise(configuration.Bitcoin)
				if nil != err {
					return err
				}
			case currency.Litecoin:
				err := litecoin.Initialise(configuration.Litecoin)
				if nil != err {
					return err
				}
			default: // only fails if new module not correctly installed
				logger.Panicf("missing payment initialiser for Currency: %s", c.String())
			}
		}
	}

	return nil
}

// stop all payment handlers
func Finalise() {
	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	if globalData.useDiscovery {
		globalData.background.Stop()
	} else {
		for c := currency.First; c <= currency.Last; c++ {
			switch c {
			case currency.Bitcoin:
				bitcoin.Finalise()
			case currency.Litecoin:
				litecoin.Finalise()
			default: // only fails if new module not correctly installed
				logger.Panicf("missing payment finaliser for Currency: %s", c.String())
			}
		}
	}

	globalData.log.Info("finished")
	globalData.log.Flush()
}
