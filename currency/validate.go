// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"unicode/utf8"

	"github.com/bitmark-inc/bitmarkd/currency/bitcoin"
	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

const (
	maxPaymentAddressLength = 64
)

// ValidateAddress - generic validate function
func (currency Currency) ValidateAddress(address string, testnet bool) error {
	if utf8.RuneCountInString(address) > maxPaymentAddressLength {
		return fault.PaymentAddressTooLong
	}

	switch currency {

	case Nothing:
		return nil // for genesis blocks

	case Bitcoin:
		version, _, err := bitcoin.ValidateAddress(address)
		if err != nil {
			return err
		}
		if bitcoin.IsTestnet(version) != testnet {
			return fault.BitcoinAddressForWrongNetwork
		}
		return nil

	case Litecoin:
		version, _, err := litecoin.ValidateAddress(address)
		if err != nil {
			return err
		}
		if litecoin.IsTestnet(version) != testnet {
			return fault.LitecoinAddressForWrongNetwork
		}
		return nil

	default:
		logger.Panicf("missing validation routine for currency: %s", currency)
	}
	return fault.InvalidCurrency
}
