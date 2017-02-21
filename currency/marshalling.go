// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

// convert a currency into JSON
func (currency Currency) MarshalText() ([]byte, error) {
	return []byte(currency.String()), nil
}

// convert currency string to a currency enumeration value from JSON
func (currency *Currency) UnmarshalText(s []byte) error {
	c, err := fromString(string(s))
	if nil != err {
		return err
	}
	*currency = c
	return nil
}
