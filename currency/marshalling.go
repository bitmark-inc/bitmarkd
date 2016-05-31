// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// convert a currency into JSON
func (currency Currency) MarshalJSON() ([]byte, error) {
	s := currency.String()
	size := 2 + len(s)
	buffer := make([]byte, size)
	buffer[0] = '"'
	buffer[size-1] = '"'
	copy(buffer[1:], s)
	return buffer, nil
}

// convert currency string to a currency enumeration value from JSON
func (currency *Currency) UnmarshalJSON(s []byte) error {
	// length = '"' + characters + '"'
	last := len(s) - 1
	if '"' != s[0] || '"' != s[last] {
		return fault.ErrInvalidCharacter
	}
	c, err := fromString(string(s[1:last]))
	if nil != err {
		return err
	}
	*currency = c
	return nil
}

// func (currency Currency) MarshalText() ([]byte, error) {
// }

// func (currency *Currency) UnmarshalText(s []byte) error {
// }
