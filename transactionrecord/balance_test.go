// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
)

// test the packing/unpacking of Share balance record
//
// ensures that pack->unpack returns the same original value
func TestPackShareBalanceOne(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("79a67be2b3d313bd490363fb0d27901c46ed53d3f7b21f60d48bc42439b06084", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkShare{
		Link:     link,
		Quantity: 12345,
	}

	expected := []byte{
		0x08, 0x20, 0x79, 0xa6, 0x7b, 0xe2, 0xb3, 0xd3,
		0x13, 0xbd, 0x49, 0x03, 0x63, 0xfb, 0x0d, 0x27,
		0x90, 0x1c, 0x46, 0xed, 0x53, 0xd3, 0xf7, 0xb2,
		0x1f, 0x60, 0xd4, 0x8b, 0xc4, 0x24, 0x39, 0xb0,
		0x60, 0x84, 0xb9, 0x60,
	}

	expectedTxId := merkle.Digest{
		0x68, 0x95, 0x6b, 0x9a, 0x91, 0x0f, 0xaa, 0x55,
		0xf3, 0x3a, 0xcb, 0xa1, 0x17, 0x08, 0x6c, 0x2f,
		0x2d, 0x83, 0x7c, 0xba, 0x9f, 0x80, 0x79, 0x87,
		0x2a, 0x4e, 0xeb, 0x65, 0x6a, 0x42, 0xeb, 0x83,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerOne.privateKey, expected)
	r.Signature = signature
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(ownerOneAccount)
	if nil != err {
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
		t.Errorf("pack txId: %#v  expected: %x", txId, expectedTxId)
		t.Errorf("*** GENERATED txId:\n%s", util.FormatBytes("expectedTxId", txId[:]))
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

	balance, ok := unpacked.(*transactionrecord.BitmarkShare)
	if !ok {
		t.Fatalf("did not unpack to BitmarkShare")
	}

	// display a JSON version for information
	item := struct {
		TxId         merkle.Digest
		ShareBalance *transactionrecord.BitmarkShare
	}{
		txId,
		balance,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Share Balance: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *balance) {
		t.Fatalf("different, original: %v  recovered: %v", r, *balance)
	}
}

// test the packing/unpacking of Share balance record
//
// ensures that zero value fails
func TestPackShareBalanceValueNotZero(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("79a67be2b3d313bd490363fb0d27901c46ed53d3f7b21f60d48bc42439b06084", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkShare{
		Link:     link,
		Quantity: 0,
	}

	expected := []byte{
		0x08, 0x20, 0x79, 0xa6, 0x7b, 0xe2, 0xb3, 0xd3,
		0x13, 0xbd, 0x49, 0x03, 0x63, 0xfb, 0x0d, 0x27,
		0x90, 0x1c, 0x46, 0xed, 0x53, 0xd3, 0xf7, 0xb2,
		0x1f, 0x60, 0xd4, 0x8b, 0xc4, 0x24, 0x39, 0xb0,
		0x60, 0x84, 0x00,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerOne.privateKey, expected)
	r.Signature = signature

	// test the packer
	_, err = r.Pack(ownerOneAccount)
	if fault.ErrShareQuantityTooSmall != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}
