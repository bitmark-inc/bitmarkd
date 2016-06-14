// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment_test

import (
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/payment"
	"testing"
)

func TestPayId(t *testing.T) {

	somedata := []byte("abcdefghijklmnopqrstuvwxyz")
	expected := `"fed399d2217aaf4c717ad0c5102c15589e1c990cc2b9a5029056a7f7485888d6ab65db2370077a5cadb53fc9280d278f"`

	payId := payment.NewPayId(somedata)
	t.Logf("pay id: %#v", payId)

	buffer, err := json.Marshal(payId)
	if nil != err {
		t.Fatalf("marshal JSON error: %v", err)
	}

	t.Logf("pay id: %s", buffer)

	actual := string(buffer)
	if expected != actual {
		t.Fatalf("pay id expected: %#v  actual: %#v", expected, actual)
	}

	var payId2 payment.PayId
	err = json.Unmarshal(buffer, &payId2)
	if nil != err {
		t.Fatalf("unmarshal JSON error: %v", err)
	}

	if payId != payId2 {
		t.Fatalf("pay id expected: %#v  actual: %#v", payId, payId2)
	}
}

func TestPayNonce(t *testing.T) {

	nonce := payment.PayNonce{0x2b, 0xa1, 0x54, 0x14, 0x46, 0x74, 0x29, 0x1d}
	expected := `"2ba154144674291d"`

	t.Logf("pay nonce: %#v", nonce)

	buffer, err := json.Marshal(nonce)
	if nil != err {
		t.Fatalf("marshal JSON error: %v", err)
	}

	t.Logf("pay nonce: %s", buffer)

	actual := string(buffer)
	if expected != actual {
		t.Fatalf("pay nonce expected: %#v  actual: %#v", expected, actual)
	}

	var nonce2 payment.PayNonce
	err = json.Unmarshal(buffer, &nonce2)
	if nil != err {
		t.Fatalf("unmarshal JSON error: %v", err)
	}

	if nonce != nonce2 {
		t.Fatalf("pay once expected: %#v  actual: %#v", nonce, nonce2)
	}
}
