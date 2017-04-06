// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration_test

import (
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"testing"
)

// test Marshal and Unmarshal
func TestSalt(t *testing.T) {
	salt, err := configuration.MakeSalt()
	if nil != err {
		t.Errorf("makeSalt fail: %s", err)
	}

	t.Logf("salt: %v\n", salt)

	marshalSalt := salt.MarshalText()

	t.Logf("salt: %q\n", marshalSalt)

	salt2 := new(configuration.Salt)
	salt2.UnmarshalText(marshalSalt)

	if salt.String() != salt2.String() {
		t.Errorf("unmarshal failed, %s != %s\n", salt.String(), salt2.String())
	}

}
