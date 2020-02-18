// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"testing"
)

func TestPrivateKeyDecodeEncode(t *testing.T) {
	originKey := "080112406eb84a3845d33c2a389d7fbea425cbf882047a2ab13084562f06875db47b5fdc2e45a298e6cd0472eeb97cd023c723824e157869d81039794864987c05b212a8"
	k, err := DecodePrivKeyFromHex(originKey)
	if nil != err {
		t.Fatalf("decode key error: %s", err)
	}
	if nil == k {
		t.Fatal("decode returned nil")
	}

	revertKey, err := EncodePrivKeyToHex(k)
	if nil != err {
		t.Fatalf("encode key error: %s", err)
	}

	if originKey != revertKey {
		t.Errorf("expected: %s  actual: %s", originKey, revertKey)
	}
}

func TestPrivateKeyDecodeEncodeWithWhitespaces(t *testing.T) {
	originKey := "080112406eb84a3845d33c2a389d7fbea425cbf882047a2ab13084562f06875db47b5fdc2e45a298e6cd0472eeb97cd023c723824e157869d81039794864987c05b212a8"
	k, err := DecodePrivKeyFromHex("   \t" + originKey + "  \n")
	if nil != err {
		t.Fatalf("decode key error: %s", err)
	}

	if nil == k {
		t.Fatal("decode returned nil")
	}

	revertKey, err := EncodePrivKeyToHex(k)
	if nil != err {
		t.Fatalf("encode key error: %s", err)
	}

	if originKey != revertKey {
		t.Errorf("expected: %s  actual: %s", originKey, revertKey)
	}
}

func TestPrivateKeyDecodeEncodeWithInvalidHex(t *testing.T) {
	originKey := "080112x06eb84a3845d33c2a389d7fbea425cbf882047a2ab13084562f06875db47b5fdc2e45a298e6cd0472eeb97cd023c723824e157869d81039794864987c05b212a8"
	k, err := DecodePrivKeyFromHex(originKey)
	if nil == err {
		t.Fatal("decode key unexpected success")
	}

	if nil != k {
		t.Errorf("decode returned: %02x", k)
	}
}

func TestNoDuplicates(t *testing.T) {

	previous := make(map[string]struct{})

	for i := 0; i < 10; i += 1 {
		pk, err := MakeEd25519PeerKey()
		if nil != err {
			t.Fatalf("encode key error: %s", err)
		}
		t.Logf("generated: %s", pk)
		_, ok := previous[pk]
		if ok {
			t.Fatalf("duplicate key generated: %s", pk)
		}
		previous[pk] = struct{}{}
	}
}
