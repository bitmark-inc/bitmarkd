// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/payment/bitcoin"
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
	Bitcoin  *bitcoin.Configuration  `libucl:"bitcoin" hcl:"bitcoin" json:"bitcoin"`
	Litecoin *litecoin.Configuration `libucl:"litecoin" hcl:"litecoin" json:"litecoin"`
}

// globals
type globalDataType struct {
	log *logger.L
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

	// initialise all currency handlers
	for c := currency.First; c <= currency.Last; c += 1 {
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

	return nil
}

// stop all payment handlers
func Finalise() {

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// finalise all currency handlers
	for c := currency.First; c <= currency.Last; c += 1 {
		switch c {
		case currency.Bitcoin:
			bitcoin.Finalise()
		case currency.Litecoin:
			litecoin.Finalise()
		default: // only fails if new module not correctly installed
			logger.Panicf("missing payment finaliser for Currency: %s", c.String())
		}
	}

	globalData.log.Info("finished")
	globalData.log.Flush()
}
