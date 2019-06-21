// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package satoshi

import (
	"testing"
)

// check the address conversion to string
func TestStringToSatoshi(t *testing.T) {
	tests := []struct {
		btc     string
		satoshi uint64
	}{
		{"", 0},
		{"0", 0},
		{"0.0", 0},
		{"0.000000001", 0},
		{"0.00000001", 1},
		{"1", 100000000},
		{"1.0", 100000000},
		{"1.00", 100000000},
		{"1.000", 100000000},
		{"1.0000", 100000000},
		{"1.00000", 100000000},
		{"1.000000", 100000000},
		{"1.0000000", 100000000},
		{"1.00000000", 100000000},
		{"1.10000000", 110000000},
		{"1.1000000", 110000000},
		{"1.100000", 110000000},
		{"1.10000", 110000000},
		{"1.1000", 110000000},
		{"1.100", 110000000},
		{"1.10", 110000000},
		{"1.1", 110000000},
		{"1.01", 101000000},
		{"1.001", 100100000},
		{"1.0001", 100010000},
		{"1.00001", 100001000},
		{"1.000001", 100000100},
		{"1.0000001", 100000010},
		{"1.00000001", 100000001},
		{"1.99999999", 199999999},
		{"9.99999999", 999999999},
		{"99999999.99999998", 9999999999999998},
		{"99999999.99999999", 9999999999999999},
	}

	for i, item := range tests {
		s := FromByteString([]byte(item.btc))
		if item.satoshi != s {
			t.Errorf("%d: BTC: %q → %d  expected: %d", i, item.btc, s, item.satoshi)
		}
	}
}
