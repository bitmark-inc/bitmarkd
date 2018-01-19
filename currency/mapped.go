// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// currency mapping to address strings
type Map map[Currency]string

// validate and pack a currency → address mapping
// create packed data as: (N = Address length)
//   MapSize              {count of key/value pairs}
//   Currency N Address   {first item}
//   …                    {more items}
//   Currency N Address   {final item}
func (m Map) Pack(testnet bool) ([]byte, error) {
	buffer := util.ToVarint64(uint64(len(m)))
	for currency, address := range m {

		if currency < First || currency > Last {
			return nil, fault.ErrInvalidCurrency
		}
		err := currency.ValidateAddress(address, testnet)
		if nil != err {
			return nil, err
		}

		buffer = append(buffer, util.ToVarint64(currency.Uint64())...)
		l := util.ToVarint64(uint64(len(address)))
		buffer = append(buffer, l...)
		buffer = append(buffer, address...)
	}
	return buffer, nil
}

// unpack and validate a currency address mapping
func UnpackMap(buffer []byte, testnet bool) (Map, Set, int, error) {

	if nil == buffer || len(buffer) < 2 {
		return nil, Set{}, 0, fault.ErrInvalidBuffer
	}

	m := make(map[Currency]string)
	cs := MakeSet()

	count, countLength := util.FromVarint64(buffer)
	n := countLength

	if count > 255 {
		return nil, Set{}, 0, fault.ErrInvalidCount
	}

	for i := 0; i < int(count); i += 1 {

		// currency
		c, currencyLength := util.FromVarint64(buffer[n:])
		currency, err := FromUint64(c)
		if nil != err {
			return nil, Set{}, 0, err
		}
		// do not allow the empty value
		if currency == Nothing {
			return nil, Set{}, 0, fault.ErrInvalidCurrency
		}

		cs.Add(currency)

		n += currencyLength

		// paymentAddress
		paymentAddressLength, paymentAddressOffset := util.FromVarint64(buffer[n:])
		n += paymentAddressOffset

		if paymentAddressLength > 255 {
			return nil, Set{}, 0, fault.ErrInvalidCount
		}

		l := int(paymentAddressLength)
		paymentAddress := string(buffer[n : n+l])
		n += int(paymentAddressLength)

		err = currency.ValidateAddress(paymentAddress, testnet)
		if nil != err {
			return nil, Set{}, 0, err
		}

		m[currency] = paymentAddress
	}

	return m, cs, n, nil
}
