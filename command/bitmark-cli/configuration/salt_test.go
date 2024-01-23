// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"testing"
)

// test Marshal and Unmarshal
func TestSalt(t *testing.T) {
	salt, err := MakeSalt()
	if err != nil {
		t.Errorf("makeSalt fail: %s", err)
	}

	//t.Logf("salt: %s\n", salt) // enable for debugging

	marshalSalt := salt.MarshalText()

	//t.Logf("salt: %s\n", marshalSalt) // enable for debugging

	salt2 := new(Salt)
	salt2.UnmarshalText(marshalSalt)

	if salt.String() != salt2.String() {
		t.Errorf("unmarshal failed, %s != %s\n", salt.String(), salt2.String())
	}

}
