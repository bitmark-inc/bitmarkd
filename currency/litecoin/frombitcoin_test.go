// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package litecoin_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/currency/litecoin"
)

// for testing
type convertAddress struct {
	btc     string
	ltc     string
	testnet bool
}

func TestConvert(t *testing.T) {

	// from: https://en.bitcoin.it/wiki/List_of_address_prefixes
	addresses := []convertAddress{
		{
			btc:     "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
			ltc:     "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
			testnet: true,
		},
		{
			btc:     "2MzQwSSnBHWHqSAqtTVQ6v47XtaisrJa1Vc",
			ltc:     "2MzQwSSnBHWHqSAqtTVQ6v47XtaisrJa1Vc",
			testnet: true,
		},
		{
			btc:     "2N1jDsGRZPMATmQYXPH7Q5QeCFqk46eTyDA",
			ltc:     "2N1jDsGRZPMATmQYXPH7Q5QeCFqk46eTyDA",
			testnet: true,
		},
		{
			btc: "17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem",
			ltc: "LRiWdjKGSjcwaNpdaPxEgcKQTpQzuT5g6d",
		},
		{
			btc: "3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQX",
			ltc: "3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQX",
		},
	}

	for i, item := range addresses {
		actualLtc, err := litecoin.FromBitcoin(item.btc)
		if err != nil {
			t.Fatalf("%d: error: %s", i, err)
		}
		if actualLtc != item.ltc {
			t.Errorf("%d: actual litecoin: %q expected: %q", i, actualLtc, item.ltc)
		}

		ltcVersion, _, err := litecoin.ValidateAddress(actualLtc)
		if err != nil {
			t.Fatalf("%d: verify error: %s", i, err)
		}

		if litecoin.IsTestnet(ltcVersion) {
			if !item.testnet {
				t.Fatalf("%d: item is not testnet", i)
			}
		} else {
			if item.testnet {
				t.Fatalf("%d: item is not livenet", i)
			}
		}

		t.Logf("%d: btc: %q ltc: %q", i, item.btc, actualLtc)

	}
}
