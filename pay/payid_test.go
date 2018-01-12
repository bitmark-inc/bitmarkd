// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pay_test

import (
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/pay"
	"testing"
)

func TestPayId(t *testing.T) {

	somedata := [][]byte{
		[]byte("abcdefghijklm"),
		[]byte("nopqrstuvwxyz"),
	}
	expected := `"fed399d2217aaf4c717ad0c5102c15589e1c990cc2b9a5029056a7f7485888d6ab65db2370077a5cadb53fc9280d278f"`

	payId := pay.NewPayId(somedata)
	t.Logf("pay id: %#v", payId)

	buffer, err := json.Marshal(payId)
	if nil != err {
		t.Fatalf("marshal JSON error: %s", err)
	}

	t.Logf("pay id: %s", buffer)

	actual := string(buffer)
	if expected != actual {
		t.Fatalf("pay id expected: %#v  actual: %#v", expected, actual)
	}

	var payId2 pay.PayId
	err = json.Unmarshal(buffer, &payId2)
	if nil != err {
		t.Fatalf("unmarshal JSON error: %s", err)
	}

	if payId != payId2 {
		t.Fatalf("pay id expected: %#v  actual: %#v", payId, payId2)
	}
}
