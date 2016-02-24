// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// limits on currency type string length
const (
	minimumCurrencyLength = 0
	maximumCurrencyLength = 16
	minimumAddressLength  = 0
	maximumAddressLength  = 64
)

// to hold miner address and its corresponding currency code
type MinerAddress struct {
	Currency string `json:"currency"`
	Address  string `json:"address"`
}

// convert a miner address to a string for use as a map index
func (m *MinerAddress) String() string {
	lc := len(m.Currency)
	if lc > maximumCurrencyLength || lc < minimumCurrencyLength {
		fault.Panic("currency string out of range")
	}
	la := len(m.Address)
	if la > maximumAddressLength || la < minimumAddressLength {
		fault.Panic("address string out of range")
	}
	return string(lc) + m.Currency + string(la) + m.Address
}

// convert and validate little endian binary byte slice to a MinerAddress
func MinerAddressFromBytes(m *MinerAddress, buffer []byte) error {

	l := len(buffer)
	if l < 2 {
		return fault.ErrCannotDecodeAddress
	}

	currencyLength := int(buffer[0])
	if l-1 <= currencyLength {
		return fault.ErrCannotDecodeAddress
	}
	if 0 == currencyLength {
		m.Currency = ""
	} else {
		m.Currency = string(buffer[1 : currencyLength+1])
	}

	i := currencyLength + 1
	l -= i
	addressLength := int(buffer[i])

	if l-1 != addressLength {
		return fault.ErrCannotDecodeAddress
	}
	if 0 == addressLength {
		m.Address = ""
	} else {
		m.Address = string(buffer[i+1:])
	}

	return nil
}
