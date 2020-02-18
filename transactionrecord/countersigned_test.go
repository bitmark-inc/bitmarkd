// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
)

// test the packing/unpacking of Bitmark transfer record
//
// transfer from issue
// ensures that pack->unpack returns the same original value
func TestPackBitmarkTransferCountersignedOne(t *testing.T) {

	issuerAccount := makeAccount(issuer.publicKey)
	ownerOneAccount := makeAccount(ownerOne.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("79a67be2b3d313bd490363fb0d27901c46ed53d3f7b21f60d48bc42439b06084", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkTransferCountersigned{
		Link:  link,
		Owner: ownerOneAccount,
	}

	expected := []byte{
		0x05, 0x20, 0x79, 0xa6, 0x7b, 0xe2, 0xb3, 0xd3,
		0x13, 0xbd, 0x49, 0x03, 0x63, 0xfb, 0x0d, 0x27,
		0x90, 0x1c, 0x46, 0xed, 0x53, 0xd3, 0xf7, 0xb2,
		0x1f, 0x60, 0xd4, 0x8b, 0xc4, 0x24, 0x39, 0xb0,
		0x60, 0x84, 0x00, 0x21, 0x13, 0x27, 0x64, 0x0e,
		0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a, 0x2f,
		0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8, 0x66,
		0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1, 0x66,
		0xb8, 0x89, 0x44, 0xb0, 0x92,
	}

	expectedTxId := merkle.Digest{
		0x6e, 0x72, 0xbb, 0x9a, 0x58, 0x50, 0xcc, 0x28,
		0x8e, 0x3c, 0x72, 0x6f, 0xbb, 0xe4, 0x33, 0xa0,
		0xe0, 0x81, 0x78, 0xac, 0xde, 0x1c, 0x8c, 0xb0,
		0x31, 0xa4, 0x36, 0x2d, 0x77, 0xa0, 0x4e, 0x09,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(issuer.privateKey, expected)
	r.Signature = signature
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// manually countersign the record and attach countersignature to "expected"
	signature = ed25519.Sign(ownerOne.privateKey, expected)
	r.Countersignature = signature
	l = util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(issuerAccount)
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

	bmt, ok := unpacked.(*transactionrecord.BitmarkTransferCountersigned)
	if !ok {
		t.Fatalf("did not unpack to BitmarkTransferCountersigned")
	}

	// display a JSON version for information
	item := struct {
		TxId                         merkle.Digest
		BitmarkTransferCountersigned *transactionrecord.BitmarkTransferCountersigned
	}{
		txId,
		bmt,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Bitmark Transfer: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *bmt) {
		t.Fatalf("different, original: %v  recovered: %v", r, *bmt)
	}
}

// test the packing/unpacking of Bitmark transfer record
//
// test transfer to transfer
// ensures that pack->unpack returns the same original value
func TestPackBitmarkTransferCountersignedTwo(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkTransferCountersigned{
		Link: link,
		Escrow: &transactionrecord.Payment{
			Currency: currency.Bitcoin,
			Address:  "mnnemVbQECtikaGZPYux4dGHH3YZyCg4sq",
			Amount:   250000,
		},
		Owner: ownerTwoAccount,
	}

	expected := []byte{
		0x05, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
		0x86, 0xbc, 0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89,
		0x18, 0x5c, 0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf,
		0xc2, 0xf0, 0x62, 0x67, 0x24, 0xf5, 0x42, 0x04,
		0x78, 0x73, 0x01, 0x01, 0x22, 0x6d, 0x6e, 0x6e,
		0x65, 0x6d, 0x56, 0x62, 0x51, 0x45, 0x43, 0x74,
		0x69, 0x6b, 0x61, 0x47, 0x5a, 0x50, 0x59, 0x75,
		0x78, 0x34, 0x64, 0x47, 0x48, 0x48, 0x33, 0x59,
		0x5a, 0x79, 0x43, 0x67, 0x34, 0x73, 0x71, 0x90,
		0xa1, 0x0f, 0x21, 0x13, 0xa1, 0x36, 0x32, 0xd5,
		0x42, 0x5a, 0xed, 0x3a, 0x6b, 0x62, 0xe2, 0xbb,
		0x6d, 0xe4, 0xc9, 0x59, 0x48, 0x41, 0xc1, 0x5b,
		0x70, 0x15, 0x69, 0xec, 0x99, 0x99, 0xdc, 0x20,
		0x1c, 0x35, 0xf7, 0xb3,
	}

	expectedTxId := merkle.Digest{
		0x2a, 0xf9, 0xd9, 0xf1, 0xbb, 0x8d, 0x8a, 0x54,
		0x68, 0x55, 0x7c, 0x86, 0x97, 0x96, 0x01, 0x18,
		0x48, 0x2c, 0x08, 0x6a, 0x76, 0xc1, 0xcb, 0xb0,
		0x21, 0xf7, 0x3f, 0x4d, 0x27, 0x65, 0x37, 0x70,
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

	bmt, ok := unpacked.(*transactionrecord.BitmarkTransferCountersigned)
	if !ok {
		t.Fatalf("did not unpack to BitmarkTransferCountersigned")
	}

	// display a JSON version for information
	item := struct {
		TxId                         merkle.Digest
		BitmarkTransferCountersigned *transactionrecord.BitmarkTransferCountersigned
	}{
		txId,
		bmt,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Bitmark Transfer: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *bmt) {
		t.Fatalf("different, original: %v  recovered: %v", r, *bmt)
	}
}

// test the packing/unpacking of Bitmark transfer record
//
// test transfer to transfer
// ensures that pack->unpack returns the same original value
func TestPackBitmarkTransferCountersignedThree(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("14eb103a0c8fb22e50e73ae9b4ff88595b1cd5f60c4afb690d8fbd014c3ed091", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkTransferCountersigned{
		Link:   link,
		Escrow: nil,
		Owner:  ownerOneAccount,
	}

	expected := []byte{
		0x05, 0x20, 0x14, 0xeb, 0x10, 0x3a, 0x0c, 0x8f,
		0xb2, 0x2e, 0x50, 0xe7, 0x3a, 0xe9, 0xb4, 0xff,
		0x88, 0x59, 0x5b, 0x1c, 0xd5, 0xf6, 0x0c, 0x4a,
		0xfb, 0x69, 0x0d, 0x8f, 0xbd, 0x01, 0x4c, 0x3e,
		0xd0, 0x91, 0x00, 0x21, 0x13, 0x27, 0x64, 0x0e,
		0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a, 0x2f,
		0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8, 0x66,
		0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1, 0x66,
		0xb8, 0x89, 0x44, 0xb0, 0x92,
	}

	expectedTxId := merkle.Digest{
		0xb9, 0xdd, 0xe2, 0x87, 0x3a, 0x98, 0x21, 0xaa,
		0x27, 0x52, 0x13, 0x76, 0x91, 0x4c, 0x8c, 0xb1,
		0x17, 0x9d, 0xb6, 0x36, 0xd3, 0x72, 0xaa, 0x4a,
		0x2a, 0x25, 0xfa, 0x2e, 0x15, 0x88, 0x3a, 0xb5,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerTwo.privateKey, expected)
	r.Signature = signature
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// manually countersign the record and attach countersignature to "expected"
	signature = ed25519.Sign(ownerOne.privateKey, expected)
	r.Countersignature = signature
	l = util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(ownerTwoAccount)
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

	bmt, ok := unpacked.(*transactionrecord.BitmarkTransferCountersigned)
	if !ok {
		t.Fatalf("did not unpack to BitmarkTransferCountersigned")
	}

	// display a JSON version for information
	item := struct {
		TxId                         merkle.Digest
		BitmarkTransferCountersigned *transactionrecord.BitmarkTransferCountersigned
	}{
		txId,
		bmt,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("Bitmark Transfer: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *bmt) {
		t.Fatalf("different, original: %v  recovered: %v", r, *bmt)
	}
}

// test the packing/unpacking of Bitmark transfer record
//
// check for error on incorrect countersignature
func TestPackBitmarkTransferCountersignedFail(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("14eb103a0c8fb22e50e73ae9b4ff88595b1cd5f60c4afb690d8fbd014c3ed091", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkTransferCountersigned{
		Link:   link,
		Escrow: nil,
		Owner:  ownerOneAccount,
	}

	expected := []byte{
		0x05, 0x20, 0x14, 0xeb, 0x10, 0x3a, 0x0c, 0x8f,
		0xb2, 0x2e, 0x50, 0xe7, 0x3a, 0xe9, 0xb4, 0xff,
		0x88, 0x59, 0x5b, 0x1c, 0xd5, 0xf6, 0x0c, 0x4a,
		0xfb, 0x69, 0x0d, 0x8f, 0xbd, 0x01, 0x4c, 0x3e,
		0xd0, 0x91, 0x00, 0x21, 0x13, 0x27, 0x64, 0x0e,
		0x4a, 0xab, 0x92, 0xd8, 0x7b, 0x4a, 0x6a, 0x2f,
		0x30, 0xb8, 0x81, 0xf4, 0x49, 0x29, 0xf8, 0x66,
		0x04, 0x3a, 0x84, 0x1c, 0x38, 0x14, 0xb1, 0x66,
		0xb8, 0x89, 0x44, 0xb0, 0x92,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerTwo.privateKey, expected)
	r.Signature = signature
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// manually countersign the record and attach countersignature to "expected"
	signature = ed25519.Sign(ownerTwo.privateKey, expected) // wrong signature
	r.Countersignature = signature

	// test the packer
	_, err = r.Pack(ownerTwoAccount)
	if fault.InvalidSignature == err {
		return
	}
	if nil == err {
		t.Error("unexpected pack success, should fail with invalid signature")
	} else {
		t.Errorf("pack error: %s", err)
	}

	t.Fatal("fatal error")
}

// test the pack failure on trying to use the zero public key
func TestPackBitmarkTransferCountersignedFromZeroAccount(t *testing.T) {

	ownerDeletedAccount := makeAccount(theZeroKey.publicKey)
	ownerOneAccount := makeAccount(ownerOne.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("14eb103a0c8fb22e50e73ae9b4ff88595b1cd5f60c4afb690d8fbd014c3ed091", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkTransferCountersigned{
		Link:             link,
		Escrow:           nil,
		Owner:            ownerOneAccount,
		Signature:        []byte{1, 2, 3, 4},
		Countersignature: []byte{1, 2, 3, 4},
	}

	// test the packer
	_, err = r.Pack(ownerDeletedAccount)
	if nil == err {
		t.Fatalf("pack should have failed")
	}
	if fault.InvalidOwnerOrRegistrant != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}

// test the pack failure on trying to use the zero public key
func TestPackBitmarkTransferCountersignedToZeroAccount(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerDeletedAccount := makeAccount(theZeroKey.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("14eb103a0c8fb22e50e73ae9b4ff88595b1cd5f60c4afb690d8fbd014c3ed091", &link)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}

	r := transactionrecord.BitmarkTransferCountersigned{
		Link:             link,
		Escrow:           nil,
		Owner:            ownerDeletedAccount,
		Signature:        []byte{1, 2, 3, 4},
		Countersignature: []byte{1, 2, 3, 4},
	}

	// test the packer
	_, err = r.Pack(ownerOneAccount)
	if nil == err {
		t.Fatalf("pack should have failed")
	}
	if fault.InvalidOwnerOrRegistrant != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}
