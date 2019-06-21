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

// test the packing/unpacking of Share grant record
//
// ensures that pack->unpack returns the same original value
func TestPackShareGrant(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var shareId merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &shareId)
	if nil != err {
		t.Fatalf("hex to share error: %s", err)
	}

	r := transactionrecord.ShareGrant{
		ShareId:   shareId,
		Quantity:  100,
		Owner:     ownerOneAccount,
		Recipient: ownerTwoAccount,
	}

	expected := []byte{
		0x09, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
		0x86, 0xbc, 0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89,
		0x18, 0x5c, 0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf,
		0xc2, 0xf0, 0x62, 0x67, 0x24, 0xf5, 0x42, 0x04,
		0x78, 0x73, 0x64, 0x21, 0x13, 0x27, 0x64, 0x0e,
		0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a, 0x2f,
		0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8, 0x66,
		0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1, 0x66,
		0xb8, 0x89, 0x44, 0xb0, 0x92, 0x21, 0x13, 0xa1,
		0x36, 0x32, 0xd5, 0x42, 0x5a, 0xed, 0x3a, 0x6b,
		0x62, 0xe2, 0xbb, 0x6d, 0xe4, 0xc9, 0x59, 0x48,
		0x41, 0xc1, 0x5b, 0x70, 0x15, 0x69, 0xec, 0x99,
		0x99, 0xdc, 0x20, 0x1c, 0x35, 0xf7, 0xb3, 0x00,
	}

	expectedTxId := merkle.Digest{
		0x26, 0x65, 0xce, 0x93, 0x0e, 0x8b, 0x0d, 0xe6,
		0x60, 0x58, 0x0d, 0xfa, 0xff, 0x05, 0x00, 0xb0,
		0x9d, 0xf9, 0xfc, 0xe8, 0x90, 0x93, 0x1b, 0xca,
		0x3b, 0x10, 0x1d, 0x3f, 0xb6, 0xf8, 0xa6, 0x1e,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerOne.privateKey, expected)
	r.Signature = signature
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// manually countersign the record and attach countersignature to "expected"
	signature = ed25519.Sign(ownerTwo.privateKey, expected)
	r.Countersignature = signature
	l = util.ToVarint64(uint64(len(signature)))
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

	grant, ok := unpacked.(*transactionrecord.ShareGrant)
	if !ok {
		t.Fatalf("did not unpack to ShareGrant")
	}

	// display a JSON version for information
	item := struct {
		TxId       merkle.Digest
		ShareGrant *transactionrecord.ShareGrant
	}{
		txId,
		grant,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Share Grant: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *grant) {
		t.Fatalf("different, original: %v  recovered: %v", r, *grant)
	}
}

// test the packing/unpacking of Share grant record
//
// ensures that value cannot be zero
func TestPackShareGrantValueNotZero(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var shareId merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &shareId)
	if nil != err {
		t.Fatalf("hex to share error: %s", err)
	}

	r := transactionrecord.ShareGrant{
		ShareId:   shareId,
		Quantity:  0,
		Owner:     ownerOneAccount,
		Recipient: ownerTwoAccount,
	}

	expected := []byte{
		0x09, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
		0x86, 0xbc, 0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89,
		0x18, 0x5c, 0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf,
		0xc2, 0xf0, 0x62, 0x67, 0x24, 0xf5, 0x42, 0x04,
		0x78, 0x73, 0x00, 0x21, 0x13, 0x27, 0x64, 0x0e,
		0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a, 0x2f,
		0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8, 0x66,
		0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1, 0x66,
		0xb8, 0x89, 0x44, 0xb0, 0x92, 0x21, 0x13, 0xa1,
		0x36, 0x32, 0xd5, 0x42, 0x5a, 0xed, 0x3a, 0x6b,
		0x62, 0xe2, 0xbb, 0x6d, 0xe4, 0xc9, 0x59, 0x48,
		0x41, 0xc1, 0x5b, 0x70, 0x15, 0x69, 0xec, 0x99,
		0x99, 0xdc, 0x20, 0x1c, 0x35, 0xf7, 0xb3, 0x00,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerOne.privateKey, expected)
	r.Signature = signature
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// manually countersign the record and attach countersignature to "expected"
	signature = ed25519.Sign(ownerTwo.privateKey, expected)
	r.Countersignature = signature

	// test the packer
	_, err = r.Pack(ownerOneAccount)
	if fault.ErrShareQuantityTooSmall != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}
