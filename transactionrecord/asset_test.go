// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"bytes"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"golang.org/x/crypto/ed25519"
	"reflect"
	"testing"
)

// test the packing/unpacking of registration record
//
// ensures that pack->unpack returns the same original value
func TestPackAssetData(t *testing.T) {

	setup(t)
	defer teardown(t)

	registrantAccount := makeAccount(registrant.publicKey)

	r := transactionrecord.AssetData{
		Name:        "Item's Name",
		Fingerprint: "0123456789abcdef",
		Metadata:    "description\x00Just the description",
		Registrant:  registrantAccount,
	}

	expected := []byte{
		0x02, 0x0b, 0x49, 0x74, 0x65, 0x6d, 0x27, 0x73,
		0x20, 0x4e, 0x61, 0x6d, 0x65, 0x10, 0x30, 0x31,
		0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39,
		0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x20, 0x64,
		0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69,
		0x6f, 0x6e, 0x00, 0x4a, 0x75, 0x73, 0x74, 0x20,
		0x74, 0x68, 0x65, 0x20, 0x64, 0x65, 0x73, 0x63,
		0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x21,
		0x13, 0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2,
		0x35, 0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73,
		0xf6, 0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49,
		0x7e, 0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11,
		0x01,
	}

	expectedTxId := merkle.Digest{
		0xa7, 0x4a, 0x90, 0xc2, 0xff, 0x76, 0x34, 0x7a,
		0x9d, 0x34, 0x19, 0xe9, 0x20, 0x2f, 0x02, 0xd8,
		0xff, 0x5d, 0xdd, 0xa2, 0x7c, 0xc1, 0x7b, 0xa1,
		0x71, 0xbc, 0x7c, 0x68, 0xbc, 0xc9, 0xce, 0x49,
	}

	expectedAssetId := transactionrecord.AssetIdentifier{
		0x59, 0xd0, 0x61, 0x55, 0xd2, 0x5d, 0xff, 0xdb,
		0x98, 0x27, 0x29, 0xde, 0x8d, 0xce, 0x9d, 0x78,
		0x55, 0xca, 0x09, 0x4d, 0x8b, 0xab, 0x81, 0x24,
		0xb3, 0x47, 0xc4, 0x06, 0x68, 0x47, 0x70, 0x56,
		0xb3, 0xc2, 0x7c, 0xcb, 0x7d, 0x71, 0xb5, 0x40,
		0x43, 0xd2, 0x07, 0xcc, 0xd1, 0x87, 0x64, 0x2b,
		0xf9, 0xc8, 0x46, 0x6f, 0x9a, 0x8d, 0x0d, 0xbe,
		0xfb, 0x4c, 0x41, 0x63, 0x3a, 0x7e, 0x39, 0xef,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(registrant.privateKey, expected)
	r.Signature = signature
	//t.Logf("signature: %#v", r.Signature)
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(registrantAccount)
	if nil != err {
		if nil != packed {
			t.Errorf("partial packed:\n%s", util.FormatBytes("expected", packed))
		}
		t.Fatalf("pack error: %s", err)
	}

	// if either of above fail we will have the message _without_ a signature
	if !bytes.Equal(packed, expected) {
		t.Errorf("pack record: %x  expected: %x", packed, expected)
		t.Errorf("*** GENERATED Packed:\n%s", util.FormatBytes("expected", packed))
		t.Fatal("fatal error")
	}

	// check the record type
	if transactionrecord.AssetDataTag != packed.Type() {
		t.Errorf("pack record type: %x  expected: %x", packed.Type(), transactionrecord.AssetDataTag)
	}

	t.Logf("Packed length: %d bytes", len(packed))

	// check txIds
	txId := packed.MakeLink()

	if txId != expectedTxId {
		t.Errorf("pack tx id: %#v  expected: %#v", txId, expectedTxId)
		t.Errorf("*** GENERATED tx id:\n%s", util.FormatBytes("expectedTxId", txId[:]))
	}

	// check asset id
	assetId := r.AssetId()

	if assetId != expectedAssetId {
		t.Errorf("pack asset id: %#v  expected: %#v", assetId, expectedAssetId)
		t.Errorf("*** GENERATED asset id:\n%s", util.FormatBytes("expectedAssetId", assetId[:]))
	}

	// test the unpacker
	unpacked, n, err := packed.Unpack(true)
	if nil != err {
		t.Fatalf("unpack error: %s", err)
	}
	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	reg, ok := unpacked.(*transactionrecord.AssetData)
	if !ok {
		t.Fatalf("did not unpack to AssetData")
	}

	// display a JSON version for information
	item := struct {
		TxId      merkle.Digest
		Asset     transactionrecord.AssetIdentifier
		AssetData *transactionrecord.AssetData
	}{
		TxId:      txId,
		Asset:     assetId,
		AssetData: reg,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("AssetData: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *reg) {
		t.Fatalf("different, original: %v  recovered: %v", r, *reg)
	}
}

// test the pack failure on missing name
func TestPackAssetDataWithEmptyName(t *testing.T) {

	registrantAccount := makeAccount(registrant.publicKey)

	r := transactionrecord.AssetData{
		Name:        "",
		Fingerprint: "0123456789abcdef",
		Metadata:    "description\x00Just the description",
		Registrant:  registrantAccount,
		Signature:   []byte{1, 2, 3, 4},
	}

	// test the packer
	_, err := r.Pack(registrantAccount)
	if nil == err {
		t.Fatalf("pack should have failed")
	}
	if fault.ErrNameTooShort != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}

// test the pack failure on missing fingerprint
func TestPackAssetDataWithEmptyFingerprint(t *testing.T) {

	registrantAccount := makeAccount(registrant.publicKey)

	r := transactionrecord.AssetData{
		Name:        "Item's Name",
		Fingerprint: "",
		Metadata:    "description\x00Just the description",
		Registrant:  registrantAccount,
		Signature:   []byte{1, 2, 3, 4},
	}

	// test the packer
	_, err := r.Pack(registrantAccount)
	if nil == err {
		t.Fatalf("pack should have failed")
	}
	if fault.ErrFingerprintTooShort != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}

// test the pack failure on invalid metadata
func TestPackAssetDataWithInvalidMetadata(t *testing.T) {

	registrantAccount := makeAccount(registrant.publicKey)

	r := transactionrecord.AssetData{
		Name:        "Item's Name",
		Fingerprint: "0123456789abcdef",
		Metadata:    "description,Just the description",
		Registrant:  registrantAccount,
		Signature:   []byte{1, 2, 3, 4},
	}

	// test the packer
	_, err := r.Pack(registrantAccount)
	if nil == err {
		t.Fatalf("pack should have failed")
	}
	if fault.ErrMetadataIsNotMap != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}

// test the packing/unpacking of registration record
//
// ensures that pack->unpack returns the same original value
func TestPackAssetDataWithEmptyMetadata(t *testing.T) {

	registrantAccount := makeAccount(registrant.publicKey)

	r := transactionrecord.AssetData{
		Name:        "Item's Name",
		Fingerprint: "0123456789abcdef",
		Metadata:    "",
		Registrant:  registrantAccount,
	}

	expected := []byte{
		0x02, 0x0b, 0x49, 0x74, 0x65, 0x6d, 0x27, 0x73,
		0x20, 0x4e, 0x61, 0x6d, 0x65, 0x10, 0x30, 0x31,
		0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39,
		0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x00, 0x21,
		0x13, 0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2,
		0x35, 0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73,
		0xf6, 0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49,
		0x7e, 0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11,
		0x01,
	}

	expectedTxId := merkle.Digest{
		0x5d, 0x6a, 0x50, 0xad, 0x18, 0xfd, 0x4b, 0x40,
		0x4c, 0x4d, 0x79, 0xf5, 0xb7, 0x55, 0x9b, 0xdd,
		0x53, 0xdf, 0x0d, 0x72, 0x6d, 0x8c, 0xed, 0x05,
		0x5d, 0xb9, 0x10, 0x08, 0x4f, 0x6b, 0xe9, 0xc1,
	}

	expectedAssetId := transactionrecord.AssetIdentifier{
		0x59, 0xd0, 0x61, 0x55, 0xd2, 0x5d, 0xff, 0xdb,
		0x98, 0x27, 0x29, 0xde, 0x8d, 0xce, 0x9d, 0x78,
		0x55, 0xca, 0x09, 0x4d, 0x8b, 0xab, 0x81, 0x24,
		0xb3, 0x47, 0xc4, 0x06, 0x68, 0x47, 0x70, 0x56,
		0xb3, 0xc2, 0x7c, 0xcb, 0x7d, 0x71, 0xb5, 0x40,
		0x43, 0xd2, 0x07, 0xcc, 0xd1, 0x87, 0x64, 0x2b,
		0xf9, 0xc8, 0x46, 0x6f, 0x9a, 0x8d, 0x0d, 0xbe,
		0xfb, 0x4c, 0x41, 0x63, 0x3a, 0x7e, 0x39, 0xef,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(registrant.privateKey, expected)
	r.Signature = signature
	//t.Logf("signature: %#v", r.Signature)
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(registrantAccount)
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

	// check the record type
	if transactionrecord.AssetDataTag != packed.Type() {
		t.Errorf("pack record type: %x  expected: %x", packed.Type(), transactionrecord.AssetDataTag)
	}

	// check txIds
	txId := packed.MakeLink()

	if txId != expectedTxId {
		t.Errorf("pack tx id: %#v  expected: %#v", txId, expectedTxId)
		t.Errorf("*** GENERATED tx id:\n%s", util.FormatBytes("expectedTxId", txId[:]))
	}

	// check asset id
	assetId := r.AssetId()

	if assetId != expectedAssetId {
		t.Errorf("pack asset id: %#v  expected: %#v", assetId, expectedAssetId)
		t.Errorf("*** GENERATED asset id:\n%s", util.FormatBytes("expectedAssetId", assetId[:]))
	}

	// test the unpacker
	unpacked, n, err := packed.Unpack(true)
	if nil != err {
		t.Fatalf("unpack error: %s", err)
	}
	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	reg, ok := unpacked.(*transactionrecord.AssetData)
	if !ok {
		t.Fatalf("did not unpack to AssetData")
	}

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *reg) {
		t.Fatalf("different, original: %v  recovered: %v", r, *reg)
	}
}
