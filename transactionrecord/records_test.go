// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"golang.org/x/crypto/ed25519"
	"reflect"
	"testing"
)

// to print a keypair for future tests
func TestGenerateKeypair(t *testing.T) {
	generate := false

	// generate = true // (uncomment to get a new key pair)

	if generate {
		// display key pair and fail the test
		// use the displayed values to modify data below
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if nil != err {
			t.Errorf("key pair generation error: %v", err)
			return
		}
		t.Errorf("*** GENERATED:\n%s", util.FormatBytes("publicKey", publicKey))
		t.Errorf("*** GENERATED:\n%s", util.FormatBytes("privateKey", privateKey))
		return
	}
}

// to hold a keypair for testing
type keyPair struct {
	publicKey  []byte
	privateKey []byte
}

// public/private keys from above generate

var proofedby = keyPair{
	publicKey: []byte{
		0x55, 0xb2, 0x98, 0x88, 0x17, 0xf7, 0xea, 0xec,
		0x37, 0x74, 0x1b, 0x82, 0x44, 0x71, 0x63, 0xca,
		0xaa, 0x5a, 0x9d, 0xb2, 0xb6, 0xf0, 0xce, 0x72,
		0x26, 0x26, 0x33, 0x8e, 0x5e, 0x3f, 0xd7, 0xf7,
	},
	privateKey: []byte{
		0x95, 0xb5, 0xa8, 0x0b, 0x4c, 0xdb, 0xe6, 0x1c,
		0x0f, 0x3f, 0x72, 0xcc, 0x15, 0x2d, 0x4a, 0x4f,
		0x29, 0xbc, 0xfd, 0x39, 0xc9, 0xa6, 0x7e, 0x2c,
		0x7b, 0xc6, 0xe0, 0xe1, 0x4e, 0xc7, 0xc7, 0xba,
		0x55, 0xb2, 0x98, 0x88, 0x17, 0xf7, 0xea, 0xec,
		0x37, 0x74, 0x1b, 0x82, 0x44, 0x71, 0x63, 0xca,
		0xaa, 0x5a, 0x9d, 0xb2, 0xb6, 0xf0, 0xce, 0x72,
		0x26, 0x26, 0x33, 0x8e, 0x5e, 0x3f, 0xd7, 0xf7,
	},
}

var registrant = keyPair{
	publicKey: []byte{
		0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2, 0x35,
		0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73, 0xf6,
		0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49, 0x7e,
		0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11, 0x01,
	},
	privateKey: []byte{
		0x66, 0xf5, 0x28, 0xd0, 0x2a, 0x64, 0x97, 0x3a,
		0x2d, 0xa6, 0x5d, 0xb0, 0x53, 0xea, 0xd0, 0xfd,
		0x94, 0xca, 0x93, 0xeb, 0x9f, 0x74, 0x02, 0x3e,
		0xbe, 0xdb, 0x2e, 0x57, 0xb2, 0x79, 0xfd, 0xf3,
		0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2, 0x35,
		0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73, 0xf6,
		0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49, 0x7e,
		0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11, 0x01,
	},
}

var issuer = keyPair{
	publicKey: []byte{
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	},
	privateKey: []byte{
		0xf3, 0xf7, 0xa1, 0xfc, 0x33, 0x10, 0x71, 0xc2,
		0xb1, 0xcb, 0xbe, 0x4f, 0x3a, 0xee, 0x23, 0x5a,
		0xae, 0xcc, 0xd8, 0x5d, 0x2a, 0x80, 0x4c, 0x44,
		0xb5, 0xc6, 0x03, 0xb4, 0xca, 0x4d, 0x9e, 0xc0,
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	},
}

var ownerOne = keyPair{
	publicKey: []byte{
		0x27, 0x64, 0x0e, 0x4a, 0xab, 0x92, 0xd8, 0x7b,
		0x4a, 0x6a, 0x2f, 0x30, 0xb8, 0x81, 0xf4, 0x49,
		0x29, 0xf8, 0x66, 0x04, 0x3a, 0x84, 0x1c, 0x38,
		0x14, 0xb1, 0x66, 0xb8, 0x89, 0x44, 0xb0, 0x92,
	},
	privateKey: []byte{
		0xc7, 0xae, 0x9f, 0x22, 0x32, 0x0e, 0xda, 0x65,
		0x02, 0x89, 0xf2, 0x64, 0x7b, 0xc3, 0xa4, 0x4f,
		0xfa, 0xe0, 0x55, 0x79, 0xcb, 0x6a, 0x42, 0x20,
		0x90, 0xb4, 0x59, 0xb3, 0x17, 0xed, 0xf4, 0xa1,
		0x27, 0x64, 0x0e, 0x4a, 0xab, 0x92, 0xd8, 0x7b,
		0x4a, 0x6a, 0x2f, 0x30, 0xb8, 0x81, 0xf4, 0x49,
		0x29, 0xf8, 0x66, 0x04, 0x3a, 0x84, 0x1c, 0x38,
		0x14, 0xb1, 0x66, 0xb8, 0x89, 0x44, 0xb0, 0x92,
	},
}

var ownerTwo = keyPair{
	publicKey: []byte{
		0xa1, 0x36, 0x32, 0xd5, 0x42, 0x5a, 0xed, 0x3a,
		0x6b, 0x62, 0xe2, 0xbb, 0x6d, 0xe4, 0xc9, 0x59,
		0x48, 0x41, 0xc1, 0x5b, 0x70, 0x15, 0x69, 0xec,
		0x99, 0x99, 0xdc, 0x20, 0x1c, 0x35, 0xf7, 0xb3,
	},
	privateKey: []byte{
		0x8f, 0x83, 0x3e, 0x58, 0x30, 0xde, 0x63, 0x77,
		0x89, 0x4a, 0x8d, 0xf2, 0xd4, 0x4b, 0x17, 0x88,
		0x39, 0x1d, 0xcd, 0xb8, 0xfa, 0x57, 0x22, 0x73,
		0xd6, 0x2e, 0x9f, 0xcb, 0x37, 0x20, 0x2a, 0xb9,
		0xa1, 0x36, 0x32, 0xd5, 0x42, 0x5a, 0xed, 0x3a,
		0x6b, 0x62, 0xe2, 0xbb, 0x6d, 0xe4, 0xc9, 0x59,
		0x48, 0x41, 0xc1, 0x5b, 0x70, 0x15, 0x69, 0xec,
		0x99, 0x99, 0xdc, 0x20, 0x1c, 0x35, 0xf7, 0xb3,
	},
}

// helper to make an address
func makeAccount(publicKey []byte) *account.Account {
	return &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      true,
			PublicKey: publicKey,
		},
	}
}

// asset id is converted from little endian by fmt.Sscan
// but merkle digests are big endian so brovide a little endian routine
func merkleDigestFromLE(s string, link *merkle.Digest) error {
	// convert little endian hex text into a digest
	return link.UnmarshalText([]byte(s))
}

// test the packing/unpacking of base record
//
// ensures that pack->unpack returns the same original value
func TestPackBaseData(t *testing.T) {

	proofedbyAccount := makeAccount(proofedby.publicKey)

	r := transactionrecord.BaseData{
		Currency:       currency.Nothing,
		PaymentAddress: "nulladdress",
		Owner:          proofedbyAccount,
		Nonce:          0x12345678,
	}

	expected := []byte{
		0x01, 0x00, 0x0b, 0x6e, 0x75, 0x6c, 0x6c, 0x61,
		0x64, 0x64, 0x72, 0x65, 0x73, 0x73, 0x21, 0x13,
		0x55, 0xb2, 0x98, 0x88, 0x17, 0xf7, 0xea, 0xec,
		0x37, 0x74, 0x1b, 0x82, 0x44, 0x71, 0x63, 0xca,
		0xaa, 0x5a, 0x9d, 0xb2, 0xb6, 0xf0, 0xce, 0x72,
		0x26, 0x26, 0x33, 0x8e, 0x5e, 0x3f, 0xd7, 0xf7,
		0xf8, 0xac, 0xd1, 0x91, 0x01,
	}

	expectedTxId := merkle.Digest{
		0x9e, 0xd1, 0x69, 0x58, 0x1f, 0xf3, 0x45, 0x02,
		0x46, 0xdc, 0xfe, 0x20, 0xf3, 0x76, 0xd8, 0x5d,
		0x56, 0xe3, 0x79, 0xc2, 0xe0, 0x97, 0xb9, 0x29,
		0xf5, 0x52, 0x4a, 0x3e, 0x6b, 0x18, 0xf4, 0x2c,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(proofedby.privateKey, expected)
	r.Signature = signature[:]
	//t.Logf("signature: %#v", r.Signature)
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature[:]...)

	// test the packer
	packed, err := r.Pack(proofedbyAccount)
	if nil != err {
		t.Fatalf("pack error: %v", err)
	}

	// if either of above fail we will have the message _without_ a signature
	if !bytes.Equal(packed, expected) {
		t.Errorf("pack record: %x  expected: %x", packed, expected)
		t.Errorf("*** GENERATED Packed:\n%s", util.FormatBytes("expected", packed))
		t.Fatal("fatal error")
	}

	// check the record type
	if transactionrecord.BaseDataTag != packed.Type() {
		t.Fatalf("pack record type: %x  expected: %x", packed.Type(), transactionrecord.BaseDataTag)
	}

	t.Logf("Packed length: %d bytes", len(packed))

	// check txIds
	txId := packed.MakeLink()

	if txId != expectedTxId {
		t.Errorf("pack tx id: %#v  expected: %#v", txId, expectedTxId)
		t.Errorf("*** GENERATED tx id:\n%s", util.FormatBytes("expectedTxId", txId[:]))
	}

	// =====
	// check test-network detection
	//
	// NOTE: this can only be done in the first record test since
	//       mode.Initialise may not be repeated
	if _, _, err := packed.Unpack(); err != fault.ErrWrongNetworkForPublicKey {
		t.Errorf("expected 'wrong network for public key' but got error: %v", err)
	}
	mode.Initialise(chain.Testing) // enter test mode - ONLY ALLOWED ONCE (or panic will occur
	// =====

	// test the unpacker
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %v", err)
	}

	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	baseData, ok := unpacked.(*transactionrecord.BaseData)
	if !ok {
		t.Fatalf("did not unpack to BaseData")
	}

	// display a JSON version for information
	item := struct {
		TxId     merkle.Digest
		BaseData *transactionrecord.BaseData
	}{
		TxId:     txId,
		BaseData: baseData,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %v", err)
	}

	t.Logf("BaseData: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *baseData) {
		t.Errorf("different, original: %v  recovered: %v", r, *baseData)
	}
}

// test the packing/unpacking of registration record
//
// ensures that pack->unpack returns the same original value
func TestPackAssetData(t *testing.T) {

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

	expectedAssetIndex := transactionrecord.AssetIndex{
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
	r.Signature = signature[:]
	//t.Logf("signature: %#v", r.Signature)
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature[:]...)

	// test the packer
	packed, err := r.Pack(registrantAccount)
	if nil != err {
		t.Fatalf("pack error: %v", err)
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

	// check asset index
	assetIndex := r.AssetIndex()

	if assetIndex != expectedAssetIndex {
		t.Errorf("pack asset index: %#v  expected: %#v", assetIndex, expectedAssetIndex)
		t.Errorf("*** GENERATED asset index:\n%s", util.FormatBytes("expectedAssetIndex", assetIndex[:]))
	}

	// test the unpacker
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %v", err)
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
		Asset     transactionrecord.AssetIndex
		AssetData *transactionrecord.AssetData
	}{
		TxId:      txId,
		Asset:     assetIndex,
		AssetData: reg,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %v", err)
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
		t.Fatalf("unexpected pack error: %v", err)
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
		t.Fatalf("unexpected pack error: %v", err)
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
		t.Fatalf("unexpected pack error: %v", err)
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

	expectedAssetIndex := transactionrecord.AssetIndex{
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
	r.Signature = signature[:]
	//t.Logf("signature: %#v", r.Signature)
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature[:]...)

	// test the packer
	packed, err := r.Pack(registrantAccount)
	if nil != err {
		t.Errorf("pack error: %v", err)
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

	// check asset index
	assetIndex := r.AssetIndex()

	if assetIndex != expectedAssetIndex {
		t.Errorf("pack asset index: %#v  expected: %#v", assetIndex, expectedAssetIndex)
		t.Errorf("*** GENERATED asset index:\n%s", util.FormatBytes("expectedAssetIndex", assetIndex[:]))
	}

	// test the unpacker
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %v", err)
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

// test the packing/unpacking of Bitmark issue record
//
// ensures that pack->unpack returns the same original value
func TestPackBitmarkIssue(t *testing.T) {

	issuerAccount := makeAccount(issuer.publicKey)

	var asset transactionrecord.AssetIndex
	_, err := fmt.Sscan("59d06155d25dffdb982729de8dce9d7855ca094d8bab8124b347c40668477056b3c27ccb7d71b54043d207ccd187642bf9c8466f9a8d0dbefb4c41633a7e39ef", &asset)
	if nil != err {
		t.Fatalf("hex to asset index error: %v", err)
	}

	r := transactionrecord.BitmarkIssue{
		AssetIndex: asset,
		Owner:      issuerAccount,
		Nonce:      99,
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
	r.Signature = signature[:]
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature[:]...)

	// test the packer
	packed, err := r.Pack(issuerAccount)
	if nil != err {
		t.Errorf("pack error: %v", err)
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
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %v", err)
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
		t.Fatalf("json error: %v", err)
	}

	t.Logf("Bitmark Issue: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *bmt) {
		t.Fatalf("different, original: %v  recovered: %v", r, *bmt)
	}
}

// make 10 separate issues for testing
//
// This only prints out 10 valid issue records that can be used for
// simple testing
func TestPackTenBitmarkIssues(t *testing.T) {

	issuerAccount := makeAccount(issuer.publicKey)

	var asset transactionrecord.AssetIndex
	_, err := fmt.Sscan("59d06155d25dffdb982729de8dce9d7855ca094d8bab8124b347c40668477056b3c27ccb7d71b54043d207ccd187642bf9c8466f9a8d0dbefb4c41633a7e39ef", &asset)
	if nil != err {
		t.Fatalf("hex to asset index error: %v", err)
	}

	rs := make([]*transactionrecord.BitmarkIssue, 10)
	for i := 0; i < len(rs); i += 1 {
		r := &transactionrecord.BitmarkIssue{
			AssetIndex: asset,
			Owner:      issuerAccount,
			Nonce:      uint64(i) + 1,
		}
		rs[i] = r

		partial, err := r.Pack(issuerAccount)
		if fault.ErrInvalidSignature != err {
			t.Fatalf("pack error: %v", err)
		}
		signature := ed25519.Sign(issuer.privateKey, partial)
		r.Signature = signature[:]

		_, err = r.Pack(issuerAccount)
		if nil != err {
			t.Fatalf("pack error: %v", err)
		}
	}
	// display a JSON version for information
	b, err := json.MarshalIndent(rs, "", "  ")
	if nil != err {
		t.Fatalf("json error: %v", err)
	}

	t.Logf("Bitmark Issue: JSON: %s", b)
}

// test the packing/unpacking of Bitmark transfer record
//
// transfer from issue
// ensures that pack->unpack returns the same original value
func TestPackBitmarkTransferOne(t *testing.T) {

	issuerAccount := makeAccount(issuer.publicKey)
	ownerOneAccount := makeAccount(ownerOne.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("79a67be2b3d313bd490363fb0d27901c46ed53d3f7b21f60d48bc42439b06084", &link)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	r := transactionrecord.BitmarkTransfer{
		Link:  link,
		Owner: ownerOneAccount,
	}

	expected := []byte{
		0x04, 0x20, 0x79, 0xa6, 0x7b, 0xe2, 0xb3, 0xd3,
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
		0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5, 0x86, 0xbc,
		0xb9, 0x09, 0x7e, 0x81, 0x61, 0x89, 0x18, 0x5c,
		0x1e, 0x03, 0x79, 0xf6, 0x7b, 0xbf, 0xc2, 0xf0,
		0x62, 0x67, 0x24, 0xf5, 0x42, 0x04, 0x78, 0x73,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(issuer.privateKey, expected)
	r.Signature = signature[:]
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature[:]...)

	// test the packer
	packed, err := r.Pack(issuerAccount)
	if nil != err {
		t.Errorf("pack error: %v", err)
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
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %v", err)
	}
	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	bmt, ok := unpacked.(*transactionrecord.BitmarkTransfer)
	if !ok {
		t.Fatalf("did not unpack to BitmarkTransfer")
	}

	// display a JSON version for information
	item := struct {
		TxId            merkle.Digest
		BitmarkTransfer *transactionrecord.BitmarkTransfer
	}{
		txId,
		bmt,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %v", err)
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
func TestPackBitmarkTransferTwo(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("630c041cd1f586bcb9097e816189185c1e0379f67bbfc2f0626724f542047873", &link)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	r := transactionrecord.BitmarkTransfer{
		Link: link,
		Payment: &transactionrecord.Payment{
			Currency: currency.Bitcoin,
			Address:  "mnnemVbQECtikaGZPYux4dGHH3YZyCg4sq",
			Amount:   250000,
		},
		Owner: ownerTwoAccount,
	}

	expected := []byte{
		0x04, 0x20, 0x63, 0x0c, 0x04, 0x1c, 0xd1, 0xf5,
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
		0x14, 0xeb, 0x10, 0x3a, 0x0c, 0x8f, 0xb2, 0x2e,
		0x50, 0xe7, 0x3a, 0xe9, 0xb4, 0xff, 0x88, 0x59,
		0x5b, 0x1c, 0xd5, 0xf6, 0x0c, 0x4a, 0xfb, 0x69,
		0x0d, 0x8f, 0xbd, 0x01, 0x4c, 0x3e, 0xd0, 0x91,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerOne.privateKey, expected)
	r.Signature = signature[:]
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature[:]...)

	// test the packer
	packed, err := r.Pack(ownerOneAccount)
	if nil != err {
		t.Errorf("pack error: %v", err)
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
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %v", err)
	}
	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	bmt, ok := unpacked.(*transactionrecord.BitmarkTransfer)
	if !ok {
		t.Fatalf("did not unpack to BitmarkTransfer")
	}

	// display a JSON version for information
	item := struct {
		TxId            merkle.Digest
		BitmarkTransfer *transactionrecord.BitmarkTransfer
	}{
		txId,
		bmt,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %v", err)
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
func TestPackBitmarkTransferThree(t *testing.T) {

	ownerOneAccount := makeAccount(ownerOne.publicKey)
	ownerTwoAccount := makeAccount(ownerTwo.publicKey)

	var link merkle.Digest
	err := merkleDigestFromLE("14eb103a0c8fb22e50e73ae9b4ff88595b1cd5f60c4afb690d8fbd014c3ed091", &link)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	r := transactionrecord.BitmarkTransfer{
		Link:    link,
		Payment: nil,
		Owner:   ownerOneAccount,
	}

	expected := []byte{
		0x04, 0x20, 0x14, 0xeb, 0x10, 0x3a, 0x0c, 0x8f,
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
		0x66, 0x58, 0x45, 0xd2, 0x19, 0xd4, 0x7d, 0x5a,
		0x2d, 0x45, 0x97, 0xb0, 0xb2, 0x31, 0xbc, 0x94,
		0x98, 0x28, 0x66, 0x84, 0x43, 0x27, 0xad, 0x02,
		0xf5, 0xed, 0x72, 0x60, 0x17, 0x3f, 0x0a, 0x9f,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(ownerTwo.privateKey, expected)
	r.Signature = signature[:]
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature[:]...)

	// test the packer
	packed, err := r.Pack(ownerTwoAccount)
	if nil != err {
		t.Errorf("pack error: %v", err)
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
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %v", err)
	}
	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	bmt, ok := unpacked.(*transactionrecord.BitmarkTransfer)
	if !ok {
		t.Fatalf("did not unpack to BitmarkTransfer")
	}

	// display a JSON version for information
	item := struct {
		TxId            merkle.Digest
		BitmarkTransfer *transactionrecord.BitmarkTransfer
	}{
		txId,
		bmt,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %v", err)
	}

	t.Logf("Bitmark Transfer: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *bmt) {
		t.Fatalf("different, original: %v  recovered: %v", r, *bmt)
	}
}
