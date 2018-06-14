// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
)

// test the packing/unpacking of Bitmark issue record
//
// ensures that pack->unpack returns the same original value
func TestPackBitmarkIssue(t *testing.T) {

	issuerAccount := makeAccount(issuer.publicKey)

	var assetId transactionrecord.AssetIdentifier
	_, err := fmt.Sscan("59d06155d25dffdb982729de8dce9d7855ca094d8bab8124b347c40668477056b3c27ccb7d71b54043d207ccd187642bf9c8466f9a8d0dbefb4c41633a7e39ef", &assetId)
	if nil != err {
		t.Fatalf("hex to asset id error: %s", err)
	}

	r := transactionrecord.BitmarkIssue{
		AssetId: assetId,
		Owner:   issuerAccount,
		Nonce:   99,
	}

	expected := []byte{
		0x03, 0x40, 0x59, 0xd0, 0x61, 0x55, 0xd2, 0x5d,
		0xff, 0xdb, 0x98, 0x27, 0x29, 0xde, 0x8d, 0xce,
		0x9d, 0x78, 0x55, 0xca, 0x09, 0x4d, 0x8b, 0xab,
		0x81, 0x24, 0xb3, 0x47, 0xc4, 0x06, 0x68, 0x47,
		0x70, 0x56, 0xb3, 0xc2, 0x7c, 0xcb, 0x7d, 0x71,
		0xb5, 0x40, 0x43, 0xd2, 0x07, 0xcc, 0xd1, 0x87,
		0x64, 0x2b, 0xf9, 0xc8, 0x46, 0x6f, 0x9a, 0x8d,
		0x0d, 0xbe, 0xfb, 0x4c, 0x41, 0x63, 0x3a, 0x7e,
		0x39, 0xef, 0x21, 0x13, 0x9f, 0xc4, 0x86, 0xa2,
		0x53, 0x4f, 0x17, 0xe3, 0x67, 0x07, 0xfa, 0x4b,
		0x95, 0x3e, 0x3b, 0x34, 0x00, 0xe2, 0x72, 0x9f,
		0x65, 0x61, 0x16, 0xdd, 0x7b, 0x01, 0x8d, 0xf3,
		0x46, 0x98, 0xbd, 0xc2, 0x63,
	}

	expectedTxId := merkle.Digest{
		0x79, 0xa6, 0x7b, 0xe2, 0xb3, 0xd3, 0x13, 0xbd,
		0x49, 0x03, 0x63, 0xfb, 0x0d, 0x27, 0x90, 0x1c,
		0x46, 0xed, 0x53, 0xd3, 0xf7, 0xb2, 0x1f, 0x60,
		0xd4, 0x8b, 0xc4, 0x24, 0x39, 0xb0, 0x60, 0x84,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(issuer.privateKey, expected)
	r.Signature = signature
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(issuerAccount)
	if nil != err {
		if nil != packed {
			t.Errorf("partial packed:\n%s", util.FormatBytes("expected", packed))
		}
		t.Errorf("pack error: %s", err)
	}

	// if either of above fail we will have the message _without_ a signature
	if !bytes.Equal(packed, expected) {
		t.Errorf("pack record: %x  expected: %x", packed, expected)
		t.Errorf("*** GENERATED Packed:\n%s", util.FormatBytes("expected", packed))
		t.Fatal("fatal error")
	}

	t.Logf("Packed length: %d bytes", len(packed))

	// check txId
	txId := packed.MakeLink()

	if txId != expectedTxId {
		t.Errorf("pack tx id: %#v  expected: %x", txId, expectedTxId)
		t.Errorf("*** GENERATED tx id:\n%s", util.FormatBytes("expectedTxId", txId[:]))
		t.Fatal("fatal error")
	}

	// test the unpacker
	unpacked, n, err := packed.Unpack(true)
	if nil != err {
		t.Fatalf("unpack error: %s", err)
	}
	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	bmt, ok := unpacked.(*transactionrecord.BitmarkIssue)
	if !ok {
		t.Fatalf("did not unpack to BitmarkIssue")
	}

	// display a JSON version for information
	item := struct {
		TxId         merkle.Digest
		BitmarkIssue *transactionrecord.BitmarkIssue
	}{
		txId,
		bmt,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Bitmark Issue: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *bmt) {
		t.Fatalf("different, original: %v  recovered: %v", r, *bmt)
	}
	checkPackedData(t, "issue", packed)
}

// make 10 separate issues for testing
//
// This only prints out 10 valid issue records that can be used for
// simple testing
func TestPackTenBitmarkIssues(t *testing.T) {

	issuerAccount := makeAccount(issuer.publicKey)

	var assetId transactionrecord.AssetIdentifier
	_, err := fmt.Sscan("59d06155d25dffdb982729de8dce9d7855ca094d8bab8124b347c40668477056b3c27ccb7d71b54043d207ccd187642bf9c8466f9a8d0dbefb4c41633a7e39ef", &assetId)
	if nil != err {
		t.Fatalf("hex to asset id error: %s", err)
	}

	rs := make([]*transactionrecord.BitmarkIssue, 10)
	for i := 0; i < len(rs); i += 1 {
		r := &transactionrecord.BitmarkIssue{
			AssetId: assetId,
			Owner:   issuerAccount,
			Nonce:   uint64(i) + 1,
		}
		rs[i] = r

		partial, err := r.Pack(issuerAccount)
		if fault.ErrInvalidSignature != err {
			if nil != partial {
				t.Errorf("partial packed:\n%s", util.FormatBytes("expected", partial))
			}
			t.Fatalf("pack error: %s", err)
		}
		signature := ed25519.Sign(issuer.privateKey, partial)
		r.Signature = signature

		_, err = r.Pack(issuerAccount)
		if nil != err {
			t.Fatalf("pack error: %s", err)
		}
	}
	// display a JSON version for information
	b, err := json.MarshalIndent(rs, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Bitmark Issue: JSON: %s", b)
}
