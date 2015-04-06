// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block_test

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"testing"
)

// check the address conversion to string
func TestAddressConversion(t *testing.T) {
	// bitcoin addresses
	items := []struct {
		a block.MinerAddress
		s string
	}{
		{
			s: "\x07" + "bitcoin" + "\x22" + "n1CuYF7iKoAxUicVT2CmyJTQdXTtMWTPNd",
			a: block.MinerAddress{Currency: "bitcoin", Address: "n1CuYF7iKoAxUicVT2CmyJTQdXTtMWTPNd"},
		},
		{
			s: "\x07" + "bitcoin" + "\x22" + "mgnZvJCMtSjaf9AEG7nd8hLtsVis5QfAMp",
			a: block.MinerAddress{Currency: "bitcoin", Address: "mgnZvJCMtSjaf9AEG7nd8hLtsVis5QfAMp"},
		},
		{
			s: "\x0b" + "justtesting" + "\x0e" + "!@#$%^&*()_+{}",
			a: block.MinerAddress{Currency: "justtesting", Address: "!@#$%^&*()_+{}"},
		},
		{
			s: "\x00\x19" + "justtesting!@#$%^&*()_+{}",
			a: block.MinerAddress{Currency: "", Address: "justtesting!@#$%^&*()_+{}"},
		},
	}

	// test all
	for i, item := range items {
		actual := item.a.String()
		if actual != item.s {
			t.Errorf("%d: to string: %q  got: %q  expected %q", i, item.a, actual, item.s)
		}

		var m block.MinerAddress
		err := block.MinerAddressFromBytes(&m, []byte(item.s))
		if nil != err {
			t.Fatalf("%d: convert from bytes failed, error: %v", i, err)
		}
		if m.Currency != item.a.Currency {
			t.Errorf("%d: from bytes: %q  currency: %q  expected %q", i, item.s, m.Currency, item.a.Currency)
		}
		if m.Address != item.a.Address {
			t.Errorf("%d: from bytes: %q  address: %q  expected %q", i, item.s, m.Address, item.a.Address)
		}
	}
}
