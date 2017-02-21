// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// convert the currency to a number
func (currency Currency) Uint64() uint64 {
	return uint64(currency)
}

// convert a number to a currency
func FromUint64(n uint64) (Currency, error) {
	if Currency(n) < maximumValue {
		return Currency(n), nil
	}
	return Nothing, fault.ErrInvalidCurrency
}
