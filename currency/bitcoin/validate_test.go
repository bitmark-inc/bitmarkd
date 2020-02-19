// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin_test

import (
	"encoding/hex"
	"testing"

	"github.com/bitmark-inc/bitmarkd/currency/bitcoin"
)

// for testing
type testAddress struct {
	address   string
	version   bitcoin.Version
	addrBytes string
	valid     bool
}

func TestMain(t *testing.T) {

	// from: https://en.bitcoin.it/wiki/List_of_address_prefixes
	addresses := []testAddress{
		{
			address:   "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
			version:   bitcoin.Testnet,
			addrBytes: "243f1394f44554f4ce3fd68649c19adc483ce924",
			valid:     true,
		},
		{
			address:   "2MzQwSSnBHWHqSAqtTVQ6v47XtaisrJa1Vc",
			version:   bitcoin.TestnetScript,
			addrBytes: "4e9f39ca4688ff102128ea4ccda34105324305b0",
			valid:     true,
		},
		{
			address:   "17VZNX1SN5NtKa8UQFxwQbFeFc3iqRYhem",
			version:   bitcoin.Livenet,
			addrBytes: "47376c6f537d62177a2c41c4ca9b45829ab99083",
			valid:     true,
		},
		{
			address:   "3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQX",
			version:   bitcoin.LivenetScript,
			addrBytes: "8f55563b9a19f321c211e9b9f38cdf686ea07845",
			valid:     true,
		},
		{
			address: "3EktnHQD7RiAE6uzMj2ZifT9YgRrkSgzQZ",
		},
		{
			address: "mipcBbFg9gMiCh81Kj9tqqdgoZub1ZJRfn",
		},
	}

	for i, item := range addresses {
		actualVersion, actualBytes, err := bitcoin.ValidateAddress(item.address)
		if item.valid {
			if nil != err {
				t.Fatalf("%d: error: %s", i, err)
			}
			eb, err := hex.DecodeString(item.addrBytes)
			if nil != err {
				t.Fatalf("%d: hex decode error: %s", i, err)
			}
			expectedBytes := bitcoin.AddressBytes{}
			if len(eb) != len(expectedBytes) {
				t.Fatalf("%d: hex length actual: %d expected: %d", i, len(eb), len(expectedBytes))
			}
			copy(expectedBytes[:], eb)

			if actualVersion != item.version {
				t.Errorf("%d: version mismatch actual: %d expected: %d", i, actualVersion, item.version)
			}
			if actualBytes != expectedBytes {
				t.Errorf("%d: bytes mismatch actual: %x expected: %x", i, actualBytes, expectedBytes)
			}

			t.Logf("%d: version: %d bytes: %x", i, actualVersion, actualBytes)
		} else if nil == err {
			t.Errorf("%d: unexpected success", i)
		}
	}
}
