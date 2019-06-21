// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// Map - currency mapping to address strings
type Map map[Currency]string

// Pack - validate and pack a currency → address mapping
// create packed data as: (N = Address length)
//   Currency N Address   {first item}
//   …                    {more items}
//   Currency N Address   {final item}
func (m Map) Pack(testnet bool) ([]byte, error) {
	buffer := make([]byte, 0, 40*len(m)) // approx: currency+byte_count+address
	n := 0
	// scan only the valid currencies
scan_currency:
	for currency := First; currency <= Last; currency += 1 {
		address := m[currency]
		if "" == address {
			continue scan_currency
		}
		err := currency.ValidateAddress(address, testnet)
		if nil != err {
			return nil, err
		}

		buffer = append(buffer, util.ToVarint64(currency.Uint64())...)
		l := util.ToVarint64(uint64(len(address)))
		buffer = append(buffer, l...)
		buffer = append(buffer, address...)
		n += 1
	}

	// check that all items were packed
	if len(m) != n {
		return nil, fault.ErrInvalidCurrency
	}

	return buffer, nil
}

// UnpackMap - unpack and validate a currency address mapping
func UnpackMap(buffer []byte, testnet bool) (Map, Set, error) {

	if nil == buffer || len(buffer) < 2 {
		return nil, Set{}, fault.ErrInvalidBuffer
	}

	m := make(map[Currency]string)
	cs := MakeSet()
	n := 0

	for n < len(buffer) {

		// currency
		c, currencyLength := util.FromVarint64(buffer[n:])
		if 0 == currencyLength {
			return nil, Set{}, fault.ErrInvalidCurrency
		}
		currency, err := FromUint64(c)
		if nil != err {
			return nil, Set{}, err
		}
		// do not allow the empty value
		if currency == Nothing {
			return nil, Set{}, fault.ErrInvalidCurrency
		}

		cs.Add(currency)

		n += currencyLength

		// paymentAddress (limit address length)
		paymentAddressLength, paymentAddressOffset := util.ClippedVarint64(buffer[n:], 1, 255)
		if 0 == paymentAddressOffset {
			return nil, Set{}, fault.ErrInvalidCount
		}
		n += paymentAddressOffset

		paymentAddress := string(buffer[n : n+paymentAddressLength])
		n += int(paymentAddressLength)

		err = currency.ValidateAddress(paymentAddress, testnet)
		if nil != err {
			return nil, Set{}, err
		}

		m[currency] = paymentAddress
	}

	return m, cs, nil
}
