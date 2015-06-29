// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction_test

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"testing"
)

func TestCursor(t *testing.T) {

	c1 := transaction.IndexCursor(0)

	buffer1, err := c1.MarshalText()
	if nil != err {
		t.Errorf("marshal text error: %v", err)
	}
	expected1 := []byte("AAAAAAAAAAA=")
	if !bytes.Equal(buffer1, expected1) {
		t.Errorf("marshal text: actual: %s != expected: %s", buffer1, expected1)
	}

	var c2 transaction.IndexCursor
	err = c2.UnmarshalText(buffer1)
	if nil != err {
		t.Errorf("unmarshal text error: %v", err)
	}

	buffer2, err := c1.MarshalJSON()
	if nil != err {
		t.Errorf("marshal JSON error: %v", err)
	}

	expected2 := []byte(`"AAAAAAAAAAA="`) // JSON includes quotes("")
	if !bytes.Equal(buffer2, expected2) {
		t.Errorf("marshal JSON: actual: %s != expected: %s", buffer2, expected2)
	}

	err = c2.UnmarshalJSON(buffer2)
	if nil != err {
		t.Errorf("unmarshal JSON error: %v", err)
	}

	for i := 0; i < 97896; i += 1 {
		c1.NextBytes()
		c2.NextBytes()
	}

	buffer1, err = c1.MarshalJSON()
	if nil != err {
		t.Errorf("marshal JSON error: %v", err)
	}
	buffer2, err = c2.MarshalJSON()
	if nil != err {
		t.Errorf("marshal JSON error: %v", err)
	}

	if !bytes.Equal(buffer1, buffer2) {
		t.Errorf("marshal JSON: %s != %s", buffer1, buffer2)
	}

	expected := []byte(`"AAAAAAABfmg="`)
	if !bytes.Equal(buffer1, expected) {
		t.Errorf("marshal JSON: actual: %s != expected: %s", buffer1, expected)
	}
}
