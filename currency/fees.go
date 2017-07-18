// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// convert a string to a currency
func (currency Currency) GetFee() (uint64, error) {
	switch currency {
	case Nothing:
		return 0, nil
	case Bitcoin:
		return 10000, nil
	case Litecoin:
		return 10000, nil
	default:
		return 0, fault.ErrInvalidCurrency
	}
}
