// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration_test

import (
	"github.com/bitmark-inc/go-programs/bitmark-cli/configuration"
	"testing"
)

// test ConvertIntegerToIter
func TestConvertIntegerToIter(t *testing.T) {
	testData := make(map[int][]byte)

	iter, err := configuration.MakeIter()
	if nil != err {
		t.Errorf("makeIter fail, %s", err)
	}

	intIter := iter.Integer()
	testData[intIter] = iter.Bytes()

	iter.ConvertIntegerToIter(intIter)
	for i, byteIter := range iter.Bytes() {
		if byteIter != testData[intIter][i] {
			t.Errorf("convert integer to iter fail.")
		}
	}
}
