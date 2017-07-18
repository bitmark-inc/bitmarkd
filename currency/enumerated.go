// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"fmt"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// currency enumeration
type Currency uint64

// possible currency values
const (
	Nothing      Currency = iota // this must be the first value
	Bitcoin      Currency = iota
	Litecoin     Currency = iota
	maximumValue Currency = iota // this must be the last value
	First        Currency = Nothing + 1
	Last         Currency = maximumValue - 1
	Count        int      = int(Last) // count of currencies
)

// internal conversion
func toString(c Currency) ([]byte, error) {
	switch c {
	case Nothing:
		return []byte{}, nil
	case Bitcoin:
		return []byte("BTC"), nil
	case Litecoin:
		return []byte("LTC"), nil
	default:
		return []byte{}, fault.ErrInvalidCurrency
	}
}

// convert a string to a currency
func fromString(in string) (Currency, error) {
	switch strings.ToLower(in) {
	case "":
		return Nothing, nil
	case "btc", "bitcoin":
		return Bitcoin, nil
	case "ltc", "litecoin":
		return Litecoin, nil
	default:
		return Nothing, fault.ErrInvalidCurrency
	}
}

// convert a currency to its string symbol
func (currency Currency) String() string {
	s, err := toString(currency)
	if nil != err {
		logger.Panicf("invalid currency enumeration: %d", currency)
	}
	return string(s)
}

// convert abot enum value and symbol, for debugging
func (currency Currency) GoString() string {
	return fmt.Sprintf("<Currency#%d:%q>", currency, currency.String())
}

// convert a currency string
func (currency *Currency) Scan(state fmt.ScanState, verb rune) error {
	token, err := state.Token(true, func(c rune) bool {
		if c >= '0' && c <= '9' {
			return true
		}
		if c >= 'A' && c <= 'Z' {
			return true
		}
		if c >= 'a' && c <= 'z' {
			return true
		}
		return false
	})
	if nil != err {
		return err
	}
	parsed, err := fromString(string(token))
	if nil != err {
		return err
	}

	*currency = parsed
	return nil
}

// valid currency if in range of First to Last
// None is not considered as valid
func (currency Currency) IsValid() bool {
	return currency >= First && currency <= Last
}

// convert a valid currency to a zero based array index
func (currency Currency) Index() int {
	if !currency.IsValid() {
		logger.Panicf("currency.Index: invalid currency: %d", currency)
	}
	return int(currency - First) // zero based index
}
