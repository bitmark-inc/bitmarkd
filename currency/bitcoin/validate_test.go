// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin_test

import (
	"github.com/bitmark-inc/bitmarkd/currency/bitcoin"
	"testing"
)

// for testing
type testAddress struct {
	address string
	version bitcoin.Version
	valid   bool
}

func TestMain(t *testing.T) {

	// from: https://en.bitcoin.it/wiki/List_of_address_prefixes
	addresses := []testAddress{
		{"mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn", bitcoin.Testnet, true},
		{"2MzQwSSnBHWHqSAqtTVQ6v47XtaisrJa1Vc", bitcoin.TestnetScript, true},
		{"17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem", bitcoin.Livenet, true},
		{"3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQX", bitcoin.LivenetScript, true},
		{"3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQZ", bitcoin.LivenetScript, false}, //fails
		{"mipcBbFg9gMiCh81Kj9tqqdgoZub1ZJRfn", bitcoin.Testnet, false},       //fails
	}

	for i, item := range addresses {
		v, err := bitcoin.ValidateAddress(item.address)
		if item.valid {
			if nil != err {
				t.Errorf("%d: error: %s", i, err)
			}
			t.Logf("%d: version: %d", i, v)
		} else if nil == err {
			t.Errorf("%d: unexpect success", i)
		}
	}
}
