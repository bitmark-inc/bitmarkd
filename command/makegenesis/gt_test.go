// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
)

// private key generation data
const (
	genesisKey = "Bitmark Inc. Genesis Block for Chain:"
	iterations = 2016
)

// to hold a keypair for testing
type keyPair struct {
	publicKey  []byte
	privateKey []byte
}

// helper to create the keypair
//
// by hashing a password + network name
func makeKeypair(netName string) (*keyPair, error) {
	switch netName {
	case "live", "test":
	default:
		return nil, fault.WrongNetworkForPublicKey
	}

	text := []byte(genesisKey + " " + netName)
	buffer := make([]byte, 32)
	for i := 0; i < iterations; i += 1 {
		buffer2 := sha3.Sum256(append(buffer, text...))
		buffer = buffer2[:]
	}

	publicKey, privateKey, err := ed25519.GenerateKey(bytes.NewBuffer(buffer))
	return &keyPair{
		publicKey:  publicKey,
		privateKey: privateKey,
	}, err
}

// helper to make an address
func makeAccount(netName string, publicKey []byte) (*account.Account, error) {
	testmode := false
	switch netName {
	case "live":
		testmode = false
	case "test":
		testmode = true
	default:
		return nil, fault.WrongNetworkForPublicKey
	}
	return &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      testmode,
			PublicKey: publicKey,
		},
	}, nil
}

// assemble the genesis base record
func TestMakeLive(t *testing.T) {

	setup(t, false)
	defer teardown(t)

	netName := "live"
	cur := currency.Nothing
	address := "DOWN the RABBIT hole"
	timestamp := uint64(0x56809ab7)

	proofedby, err := makeKeypair(netName)
	if err != nil {
		t.Fatalf("makeKeypair error: %s", err)
	}

	proofedbyAccount, err := makeAccount(netName, proofedby.publicKey)
	if err != nil {
		t.Fatalf("makeAccount error: %s", err)
	}

	base := &transactionrecord.OldBaseData{
		Currency:       cur,
		PaymentAddress: address,
		Owner:          proofedbyAccount,
		Nonce:          0x4c6976652a4e6574, // Live*Net
	}

	// expected block header
	hExpected := blockrecord.PackedHeader{
		0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x63, 0x8c, 0x15, 0x9c,
		0x1f, 0x11, 0x3f, 0x70, 0xa9, 0x86, 0x6d, 0x9a,
		0x9e, 0x52, 0xe9, 0xef, 0xe9, 0xb9, 0x92, 0x08,
		0x48, 0xad, 0x1d, 0xf3, 0x48, 0x51, 0xbe, 0x8a,
		0x56, 0x2a, 0x99, 0x8d, 0xb7, 0x9a, 0x80, 0x56,
		0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	bExpected := []byte{
		0x01, 0x00, 0x14, 0x44, 0x4f, 0x57, 0x4e, 0x20,
		0x74, 0x68, 0x65, 0x20, 0x52, 0x41, 0x42, 0x42,
		0x49, 0x54, 0x20, 0x68, 0x6f, 0x6c, 0x65, 0x21,
		0x11, 0x4a, 0x65, 0xf1, 0xd2, 0x06, 0x50, 0x08,
		0x12, 0x76, 0xf0, 0x1d, 0xf4, 0x3e, 0x70, 0x55,
		0x4e, 0x95, 0x49, 0x8f, 0x37, 0x78, 0xe5, 0x6d,
		0xaa, 0x2c, 0x49, 0x82, 0x03, 0xae, 0x9c, 0x70,
		0xe6, 0xf4, 0xca, 0xb9, 0xd2, 0xd2, 0xcc, 0xdd,
		0xb4, 0x4c,
	}

	expectedTxId := merkle.Digest{
		0x63, 0x8c, 0x15, 0x9c, 0x1f, 0x11, 0x3f, 0x70,
		0xa9, 0x86, 0x6d, 0x9a, 0x9e, 0x52, 0xe9, 0xef,
		0xe9, 0xb9, 0x92, 0x08, 0x48, 0xad, 0x1d, 0xf3,
		0x48, 0x51, 0xbe, 0x8a, 0x56, 0x2a, 0x99, 0x8d,
	}

	checkGenesis(t, netName, base, proofedby, timestamp, hExpected, bExpected, expectedTxId)
}

// assemble the genesis base record
func TestMakeTest(t *testing.T) {

	setup(t, true)
	defer teardown(t)

	netName := "test"
	cur := currency.Nothing
	address := "Bitmark Testing Genesis Block"
	timestamp := uint64(0x5478424b)

	proofedby, err := makeKeypair(netName)
	if err != nil {
		t.Fatalf("makeKeypair error: %s", err)
	}

	proofedbyAccount, err := makeAccount(netName, proofedby.publicKey)
	if err != nil {
		t.Fatalf("makeAccount error: %s", err)
	}

	base := &transactionrecord.OldBaseData{
		Currency:       cur,
		PaymentAddress: address,
		Owner:          proofedbyAccount,
		Nonce:          0x546573742a4e6574, // Test*Net
	}

	// expected block header
	hExpected := blockrecord.PackedHeader{
		0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0xee, 0x07, 0xbb, 0xc3,
		0xd7, 0x49, 0xe0, 0x7d, 0x24, 0xb9, 0x0c, 0xd1,
		0xec, 0x35, 0x14, 0x70, 0x2e, 0x87, 0x85, 0x22,
		0xda, 0xf7, 0x16, 0xc1, 0x73, 0x24, 0xd6, 0x66,
		0x69, 0x7b, 0x8a, 0x63, 0x4b, 0x42, 0x78, 0x54,
		0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}

	bExpected := []byte{
		0x01, 0x00, 0x1d, 0x42, 0x69, 0x74, 0x6d, 0x61,
		0x72, 0x6b, 0x20, 0x54, 0x65, 0x73, 0x74, 0x69,
		0x6e, 0x67, 0x20, 0x47, 0x65, 0x6e, 0x65, 0x73,
		0x69, 0x73, 0x20, 0x42, 0x6c, 0x6f, 0x63, 0x6b,
		0x21, 0x13, 0xb2, 0xb5, 0x04, 0x82, 0x7f, 0x30,
		0xa8, 0xdc, 0x1b, 0x75, 0x95, 0xeb, 0xb9, 0x88,
		0xdc, 0xf8, 0x7c, 0xad, 0xac, 0x9e, 0x3a, 0x38,
		0xf6, 0xbe, 0x81, 0x8c, 0x72, 0xbe, 0x03, 0x35,
		0xfa, 0x74, 0xf4, 0xca, 0xb9, 0xd2, 0xc2, 0xee,
		0xdc, 0xb2, 0x54,
	}

	expectedTxId := merkle.Digest{
		0xee, 0x07, 0xbb, 0xc3, 0xd7, 0x49, 0xe0, 0x7d,
		0x24, 0xb9, 0x0c, 0xd1, 0xec, 0x35, 0x14, 0x70,
		0x2e, 0x87, 0x85, 0x22, 0xda, 0xf7, 0x16, 0xc1,
		0x73, 0x24, 0xd6, 0x66, 0x69, 0x7b, 0x8a, 0x63,
	}

	checkGenesis(t, netName, base, proofedby, timestamp, hExpected, bExpected, expectedTxId)
}

func checkGenesis(t *testing.T, netName string, base *transactionrecord.OldBaseData, proofedby *keyPair, timestamp uint64, hExpected blockrecord.PackedHeader, bExpected []byte, expectedTxId merkle.Digest) {

	// manually sign the record and attach signature to "bExpected"
	signature := ed25519.Sign(proofedby.privateKey, bExpected)
	base.Signature = signature

	t.Logf("*** GENERATED signature:\n%s", util.FormatBytes("signature", signature))

	l := util.ToVarint64(uint64(len(signature)))
	bExpected = append(bExpected, l...)
	bExpected = append(bExpected, signature...)

	// test the packer
	packed, err := base.Pack(base.Owner)
	if err != nil {
		t.Fatalf("pack error: %s", err)
	}

	// if either of above fail we will have the message _without_ a signature
	if !bytes.Equal(packed, bExpected) {
		t.Errorf("pack record: %x  bExpected: %x", packed, bExpected)
		t.Fatalf("*** GENERATED Packed:\n%s", util.FormatBytes("bExpected", packed))
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
		t.Fatalf("*** GENERATED tx id:\n%s", util.FormatBytes("expectedTxId", txId[:]))
	}

	// test the unpacker
	unpacked, n, err := packed.Unpack(base.Owner.IsTesting())
	if err != nil {
		t.Fatalf("unpack error: %s", err)
	}
	if len(packed) != n {
		t.Fatalf("unpack error: ony read %d of %d bytes", n, len(packed))
	}

	baseData, ok := unpacked.(*transactionrecord.OldBaseData)
	if !ok {
		t.Fatalf("did not unpack to BaseData")
	}

	// display a JSON version for information
	item := struct {
		HexTxId  string
		TxId     merkle.Digest
		BaseData *transactionrecord.OldBaseData
	}{
		HexTxId:  txId.String(),
		TxId:     txId,
		BaseData: baseData,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("BaseData: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(*base, *baseData) {
		t.Fatalf("different, original: %v  recovered: %v", *base, *baseData)
	}

	// make the header

	// default difficulty
	diffy := difficulty.New() // defaults to 1
	t.Logf("difficulty New: %v", diffy)

	nonce := blockrecord.NonceType(0)

	// block header
	// PreviousBlock: []byte{0,0,...},
	h := &blockrecord.Header{
		Version:          1,
		TransactionCount: 1,
		Number:           1,
		MerkleRoot:       merkle.Digest(txId),
		Timestamp:        timestamp,
		Difficulty:       diffy,
		Nonce:            nonce,
	}

	t.Logf("%s", util.FormatBytes("merkle_root", txId[:]))

	hPacked := h.Pack()
	t.Logf("packed block header length: %d", len(hPacked))
	t.Logf("Packed block header: %x", hPacked)

	// unpack and check
	hUnpacked, err := hPacked.Unpack()
	if err != nil {
		t.Fatalf("unpack block header error: %s", err)
	}

	// check then unpacked matches original
	if !reflect.DeepEqual(h, hUnpacked) {
		t.Fatalf("different, original: %v  recovered: %v", h, hUnpacked)
	}

	// check that packed header matches
	if hPacked != hExpected {
		t.Errorf("pack record: %x  expected: %x", hPacked, hExpected)
		t.Fatalf("*** GENERATED Packed:\n%s", util.FormatBytes("hExpected", hPacked[:]))
	}

	// JSON block header
	b, err = json.MarshalIndent(hUnpacked, "", "  ")
	if err != nil {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("BlockHeader: JSON: %s", b)

}
