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

// test the packing/unpacking of Share swap record
//
// ensures that pack->unpack returns the same original value
func TestPackShareSwap(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var shareIdOne merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &shareIdOne)
	if nil != err {
		t.Fatalf("hex to shareIdOne error: %s", err)
	}

	var shareIdTwo merkle.Digest
	err = merkleDigestFromLE("79a67be2b3d313bd490363fb0d27901c46ed53d3f7b21f60d48bc42439b06084", &shareIdTwo)
	if nil != err {
		t.Fatalf("hex to shareIdTwo error: %s", err)
	}

	r := transactionrecord.ShareSwap{
		ShareIdOne:  shareIdOne,
		QuantityOne: 129,
		OwnerOne:    ownerOneAccount,
		ShareIdTwo:  shareIdTwo,
		QuantityTwo: 215,
		OwnerTwo:    ownerTwoAccount,
	}

	expected := []byte{
		0x0a, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
		0x86, 0xbc, 0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89,
		0x18, 0x5c, 0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf,
		0xc2, 0xf0, 0x62, 0x67, 0x24, 0xf5, 0x42, 0x04,
		0x78, 0x73, 0x81, 0x01, 0x21, 0x13, 0x27, 0x64,
		0x0e, 0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a,
		0x2f, 0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8,
		0x66, 0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1,
		0x66, 0xb8, 0x89, 0x44, 0xb0, 0x92, 0x20, 0x79,
		0xa6, 0x7b, 0xe2, 0xb3, 0xd3, 0x13, 0xbd, 0x49,
		0x03, 0x63, 0xfb, 0x0d, 0x27, 0x90, 0x1c, 0x46,
		0xed, 0x53, 0xd3, 0xf7, 0xb2, 0x1f, 0x60, 0xd4,
		0x8b, 0xc4, 0x24, 0x39, 0xb0, 0x60, 0x84, 0xd7,
		0x01, 0x21, 0x13, 0xa1, 0x36, 0x32, 0xd5, 0x42,
		0x5a, 0xed, 0x3a, 0x6b, 0x62, 0xe2, 0xbb, 0x6d,
		0xe4, 0xc9, 0x59, 0x48, 0x41, 0xc1, 0x5b, 0x70,
		0x15, 0x69, 0xec, 0x99, 0x99, 0xdc, 0x20, 0x1c,
		0x35, 0xf7, 0xb3, 0x00,
	}

	expectedTxId := merkle.Digest{
		0x53, 0xd9, 0x9f, 0x56, 0x51, 0xe9, 0x94, 0x3a,
		0x1d, 0x73, 0xdb, 0x57, 0xf3, 0xe0, 0x76, 0x59,
		0x4f, 0x85, 0xa8, 0x9b, 0x8a, 0xd5, 0x3e, 0xc3,
		0x87, 0x71, 0x9a, 0xde, 0x80, 0x54, 0x01, 0x62,
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

	swap, ok := unpacked.(*transactionrecord.ShareSwap)
	if !ok {
		t.Fatalf("did not unpack to ShareSwap")
	}

	// display a JSON version for information
	item := struct {
		TxId      merkle.Digest
		ShareSwap *transactionrecord.ShareSwap
	}{
		txId,
		swap,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Share Swap: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *swap) {
		t.Fatalf("different, original: %v  recovered: %v", r, *swap)
	}
}

// test the packing/unpacking of Share swap record
//
// ensures that quantity one cannot be zero
func TestPackShareSwapQuantityOneNotZero(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var shareIdOne merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &shareIdOne)
	if nil != err {
		t.Fatalf("hex to shareIdOne error: %s", err)
	}

	var shareIdTwo merkle.Digest
	err = merkleDigestFromLE("79a67be2b3d313bd490363fb0d27901c46ed53d3f7b21f60d48bc42439b06084", &shareIdTwo)
	if nil != err {
		t.Fatalf("hex to shareIdTwo error: %s", err)
	}

	r := transactionrecord.ShareSwap{
		ShareIdOne:  shareIdOne,
		QuantityOne: 0,
		OwnerOne:    ownerOneAccount,
		ShareIdTwo:  shareIdTwo,
		QuantityTwo: 215,
		OwnerTwo:    ownerTwoAccount,
	}

	expected := []byte{
		0x0a, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
		0x86, 0xbc, 0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89,
		0x18, 0x5c, 0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf,
		0xc2, 0xf0, 0x62, 0x67, 0x24, 0xf5, 0x42, 0x04,
		0x78, 0x73, 0x00, 0x21, 0x13, 0x27, 0x64, 0x0e,
		0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a, 0x2f,
		0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8, 0x66,
		0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1, 0x66,
		0xb8, 0x89, 0x44, 0xb0, 0x92, 0x20, 0x79, 0xa6,
		0x7b, 0xe2, 0xb3, 0xd3, 0x13, 0xbd, 0x49, 0x03,
		0x63, 0xfb, 0x0d, 0x27, 0x90, 0x1c, 0x46, 0xed,
		0x53, 0xd3, 0xf7, 0xb2, 0x1f, 0x60, 0xd4, 0x8b,
		0xc4, 0x24, 0x39, 0xb0, 0x60, 0x84, 0xd7, 0x01,
		0x21, 0x13, 0xa1, 0x36, 0x32, 0xd5, 0x42, 0x5a,
		0xed, 0x3a, 0x6b, 0x62, 0xe2, 0xbb, 0x6d, 0xe4,
		0xc9, 0x59, 0x48, 0x41, 0xc1, 0x5b, 0x70, 0x15,
		0x69, 0xec, 0x99, 0x99, 0xdc, 0x20, 0x1c, 0x35,
		0xf7, 0xb3, 0x00}

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
	_, err = r.Pack(ownerOneAccount)
	if fault.ErrShareQuantityTooSmall != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}

// test the packing/unpacking of Share swap record
//
// ensures that quantity one cannot be zero
func TestPackShareSwapQuantityTwoNotZero(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var shareIdOne merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &shareIdOne)
	if nil != err {
		t.Fatalf("hex to shareIdOne error: %s", err)
	}

	var shareIdTwo merkle.Digest
	err = merkleDigestFromLE("79a67be2b3d313bd490363fb0d27901c46ed53d3f7b21f60d48bc42439b06084", &shareIdTwo)
	if nil != err {
		t.Fatalf("hex to shareIdTwo error: %s", err)
	}

	r := transactionrecord.ShareSwap{
		ShareIdOne:  shareIdOne,
		QuantityOne: 129,
		OwnerOne:    ownerOneAccount,
		ShareIdTwo:  shareIdTwo,
		QuantityTwo: 0,
		OwnerTwo:    ownerTwoAccount,
	}

	expected := []byte{
		0x0a, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
		0x86, 0xbc, 0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89,
		0x18, 0x5c, 0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf,
		0xc2, 0xf0, 0x62, 0x67, 0x24, 0xf5, 0x42, 0x04,
		0x78, 0x73, 0x81, 0x01, 0x21, 0x13, 0x27, 0x64,
		0x0e, 0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a,
		0x2f, 0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8,
		0x66, 0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1,
		0x66, 0xb8, 0x89, 0x44, 0xb0, 0x92, 0x20, 0x79,
		0xa6, 0x7b, 0xe2, 0xb3, 0xd3, 0x13, 0xbd, 0x49,
		0x03, 0x63, 0xfb, 0x0d, 0x27, 0x90, 0x1c, 0x46,
		0xed, 0x53, 0xd3, 0xf7, 0xb2, 0x1f, 0x60, 0xd4,
		0x8b, 0xc4, 0x24, 0x39, 0xb0, 0x60, 0x84, 0x00,
		0x21, 0x13, 0xa1, 0x36, 0x32, 0xd5, 0x42, 0x5a,
		0xed, 0x3a, 0x6b, 0x62, 0xe2, 0xbb, 0x6d, 0xe4,
		0xc9, 0x59, 0x48, 0x41, 0xc1, 0x5b, 0x70, 0x15,
		0x69, 0xec, 0x99, 0x99, 0xdc, 0x20, 0x1c, 0x35,
		0xf7, 0xb3, 0x00}

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
	_, err = r.Pack(ownerOneAccount)
	if fault.ErrShareQuantityTooSmall != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}

// test the packing/unpacking of Share swap record
//
// ensures that shares cannot be the same
func TestPackShareSwapSharesDoNotMatch(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var shareIdOne merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &shareIdOne)
	if nil != err {
		t.Fatalf("hex to shareIdOne error: %s", err)
	}

	var shareIdTwo merkle.Digest
	err = merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &shareIdTwo)
	if nil != err {
		t.Fatalf("hex to shareIdTwo error: %s", err)
	}

	r := transactionrecord.ShareSwap{
		ShareIdOne:  shareIdOne,
		QuantityOne: 129,
		OwnerOne:    ownerOneAccount,
		ShareIdTwo:  shareIdTwo,
		QuantityTwo: 215,
		OwnerTwo:    ownerTwoAccount,
	}

	expected := []byte{
		0x0a, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
		0x86, 0xbc, 0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89,
		0x18, 0x5c, 0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf,
		0xc2, 0xf0, 0x62, 0x67, 0x24, 0xf5, 0x42, 0x04,
		0x78, 0x73, 0x81, 0x01, 0x21, 0x13, 0x27, 0x64,
		0x0e, 0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a,
		0x2f, 0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8,
		0x66, 0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1,
		0x66, 0xb8, 0x89, 0x44, 0xb0, 0x92, 0x20, 0x63,
		0x0c, 0x04, 0x1c, 0xd1, 0xf5, 0x86, 0xbc, 0xb9,
		0x09, 0x7e, 0x81, 0x61, 0x89, 0x18, 0x5c, 0x1e,
		0x03, 0x79, 0xf6, 0x7b, 0xbf, 0xc2, 0xf0, 0x62,
		0x67, 0x24, 0xf5, 0x42, 0x04, 0x78, 0x73, 0xd7,
		0x01, 0x21, 0x13, 0xa1, 0x36, 0x32, 0xd5, 0x42,
		0x5a, 0xed, 0x3a, 0x6b, 0x62, 0xe2, 0xbb, 0x6d,
		0xe4, 0xc9, 0x59, 0x48, 0x41, 0xc1, 0x5b, 0x70,
		0x15, 0x69, 0xec, 0x99, 0x99, 0xdc, 0x20, 0x1c,
		0x35, 0xf7, 0xb3, 0x00,
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
	_, err = r.Pack(ownerOneAccount)
	if fault.ErrShareIdsCannotBeIdentical != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}
