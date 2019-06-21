// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// GetFee - returns the fee for a specific currency
func (currency Currency) GetFee() (uint64, error) {
	switch currency {
	case Nothing:
		return 0, nil
	case Bitcoin:
		return 10000, nil
	case Litecoin:
		return 100000, nil // as of 2017-07-28 Litecoin penalises any Vout < 100,000 Satoshi
	default:
		return 0, fault.ErrInvalidCurrency
	}
}
