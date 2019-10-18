// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir_test

import (
	"encoding/json"
	"testing"

	"github.com/bitmark-inc/bitmarkd/chain"

	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/reservoir"
)

func TestPayNonce(t *testing.T) {

	setup(t)
	defer teardown()

	nonce := reservoir.PayNonce{
		0x2b, 0xa1, 0x54, 0x14, 0x46, 0x74, 0x29, 0x1d,
	}
	expected := `"2ba154144674291d"`

	t.Logf("pay nonce: %#v", nonce)

	buffer, err := json.Marshal(nonce)
	if nil != err {
		t.Fatalf("marshal JSON error: %s", err)
	}

	t.Logf("pay nonce: %s", buffer)

	actual := string(buffer)
	if expected != actual {
		t.Fatalf("pay nonce expected: %#v  actual: %#v", expected, actual)
	}

	var nonce2 reservoir.PayNonce
	err = json.Unmarshal(buffer, &nonce2)
	if nil != err {
		t.Fatalf("unmarshal JSON error: %s", err)
	}

	if nonce != nonce2 {
		t.Fatalf("pay once expected: %#v  actual: %#v", nonce, nonce2)
	}
}

func TestNewPayNonceBitmark(t *testing.T) {

	// dependant on the genesis digest for bitmark
	expected := `"5c93f739eb01cdde"`

	setup(t)
	defer teardown()

	d, n := blockheader.GetNew()
	t.Logf("block: %d  %#v", n, d)

	nonce := reservoir.NewPayNonce()
	t.Logf("pay nonce: %#v", nonce)

	buffer, err := json.Marshal(nonce)
	if nil != err {
		t.Fatalf("marshal JSON error: %s", err)
	}

	t.Logf("pay nonce: %s", buffer)

	actual := string(buffer)
	if expected != actual {
		t.Fatalf("pay nonce expected: %#v  actual: %#v", expected, actual)
	}
}

func TestNewPayNonceTesting(t *testing.T) {

	// dependant on the genesis digest for testing
	expected := `"8ae68bb87c4a926b"`

	setup(t, chain.Testing)
	defer teardown()

	d, n := blockheader.GetNew()
	t.Logf("block: %d  %#v", n, d)

	nonce := reservoir.NewPayNonce()
	t.Logf("pay nonce: %#v", nonce)

	buffer, err := json.Marshal(nonce)
	if nil != err {
		t.Fatalf("marshal JSON error: %s", err)
	}

	actual := string(buffer)
	if expected != actual {
		t.Fatalf("pay nonce expected: %#v  actual: %#v", expected, actual)
	}

}
