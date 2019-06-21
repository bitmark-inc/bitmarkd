// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord_test

import (
	"encoding/json"
	"testing"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
)

// test JSON conversion
func TestInitialBits(t *testing.T) {

	nonces := []blockrecord.NonceType{
		0x1234567890abcdef,
		0x1234567890abcdef,
		0x1234567890abcdef,
	}

	for i, expected := range nonces {

		buffer, err := json.Marshal(expected)
		if nil != err {
			t.Fatalf("%d: JSON encode error: %s", i, err)
		}

		var actual blockrecord.NonceType
		err = json.Unmarshal(buffer, &actual)
		if nil != err {
			t.Fatalf("%d: JSON decode error: %s", i, err)
		}

		if actual != expected {
			t.Errorf("%d: JSON actual: %016x  expected: %016x", i, actual, expected)
		}
	}
}
