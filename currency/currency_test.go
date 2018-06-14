// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package currency_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
)

type currencyTest struct {
	str string
	c   currency.Currency
	j   string
}

var valid = []currencyTest{
	{"", currency.Nothing, `""`},
	{"btc", currency.Bitcoin, `"BTC"`},
	{"BTC", currency.Bitcoin, `"BTC"`},
	{"Bitcoin", currency.Bitcoin, `"BTC"`},
	{"BITCOIN", currency.Bitcoin, `"BTC"`},
	{"BitCoin", currency.Bitcoin, `"BTC"`},
	{"bitcoin", currency.Bitcoin, `"BTC"`},
	{"ltc", currency.Litecoin, `"LTC"`},
	{"LTC", currency.Litecoin, `"LTC"`},
	{"Litecoin", currency.Litecoin, `"LTC"`},
	{"LITECOIN", currency.Litecoin, `"LTC"`},
	{"LiteCoin", currency.Litecoin, `"LTC"`},
	{"litecoin", currency.Litecoin, `"LTC"`},
}

var invalid = []string{
	"389749837598",
	"null",
	"a b",
}

func TestValidString(t *testing.T) {
	for index, test := range valid {

		var c currency.Currency
		n, err := fmt.Sscan(test.str, &c)
		if nil != err {
			t.Fatalf("%d: string to currency error: %s", index, err)
		}

		if 1 != n {
			t.Fatalf("%d: scanned %d items expected to scan 1", index, n)
		}

		if c != test.c {
			t.Errorf("%d: %q converted to: %#v  expected: %#v", index, test.str, c, test.c)
		}
	}
}

func TestInvalidString(t *testing.T) {
	for index, test := range invalid {

		var c currency.Currency
		n, err := fmt.Sscan(test, &c)
		if fault.ErrInvalidCurrency != err {
			t.Fatalf("%d: string to currency error: %s", index, err)
		}

		if 0 != n {
			t.Fatalf("%d: scanned %d items expected to scan 0(zero)", index, n)
		}

	}
}

func TestMarshalling(t *testing.T) {
	for index, test := range valid {

		buffer, err := json.Marshal(test.c)
		if nil != err {
			t.Fatalf("%d: Marshal JSON error: %s", index, err)
		}

		if test.j != string(buffer) {
			t.Errorf("%d: Marshal JSON expected: %q  actual: %q", index, test.j, buffer)
		}

	}
}

func TestUnmarshalling(t *testing.T) {
	for index, test := range valid {

		buffer := []byte(`"` + test.str + `"`)
		var c currency.Currency
		err := json.Unmarshal(buffer, &c)
		if nil != err {
			t.Fatalf("%d: Unmarshal JSON error: %s", index, err)
		}

		if test.c != c {
			t.Errorf("%d: Unmarshal JSON expected: %#v  actual: %#v", index, test.c, c)
		}

	}
}
