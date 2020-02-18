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

// test the packing/unpacking of base record
//
// ensures that pack->unpack returns the same original value
func TestPackBaseData(t *testing.T) {

	proofedByAccount := makeAccount(proofedBy.publicKey)

	r := transactionrecord.OldBaseData{
		Currency:       currency.Bitcoin,
		PaymentAddress: "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
		Owner:          proofedByAccount,
		Nonce:          0x12345678,
	}

	expected := []byte{
		0x01, 0x01, 0x22, 0x6d, 0x69, 0x70, 0x63, 0x42,
		0x62, 0x46, 0x67, 0x39, 0x67, 0x4d, 0x69, 0x43,
		0x68, 0x38, 0x31, 0x4b, 0x6a, 0x38, 0x74, 0x71,
		0x71, 0x64, 0x67, 0x6f, 0x5a, 0x75, 0x62, 0x31,
		0x5a, 0x4a, 0x52, 0x66, 0x6e, 0x21, 0x13, 0x55,
		0xb2, 0x98, 0x88, 0x17, 0xf7, 0xea, 0xec, 0x37,
		0x74, 0x1b, 0x82, 0x44, 0x71, 0x63, 0xca, 0xaa,
		0x5a, 0x9d, 0xb2, 0xb6, 0xf0, 0xce, 0x72, 0x26,
		0x26, 0x33, 0x8e, 0x5e, 0x3f, 0xd7, 0xf7, 0xf8,
		0xac, 0xd1, 0x91, 0x01,
	}

	expectedTxId := merkle.Digest{
		0x9e, 0x93, 0x2b, 0x8e, 0xa1, 0xa3, 0xd4, 0x30,
		0xc5, 0x9a, 0x23, 0xfd, 0x56, 0x75, 0xe8, 0xba,
		0x64, 0x0e, 0xe8, 0x1c, 0xf3, 0x0e, 0x68, 0xca,
		0x14, 0x8e, 0xe1, 0x1f, 0x13, 0xdb, 0xd4, 0x27,
	}

	// manually sign the record and attach signature to "expected"
	signature := ed25519.Sign(proofedBy.privateKey, expected)
	r.Signature = signature
	//t.Logf("signature: %#v", r.Signature)
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(proofedByAccount)
	if nil != err {
		t.Fatalf("pack error: %s", err)
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

	// test the unpacker
	unpacked, n, err := packed.Unpack(true)
	if nil != err {
		t.Fatalf("unpack error: %s", err)
	}

	if len(packed) != n {
		t.Errorf("did not unpack all data: only used: %d of: %d bytes", n, len(packed))
	}

	baseData, ok := unpacked.(*transactionrecord.OldBaseData)
	if !ok {
		t.Fatalf("did not unpack to BaseData")
	}

	// display a JSON version for information
	item := struct {
		TxId     merkle.Digest
		BaseData *transactionrecord.OldBaseData
	}{
		TxId:     txId,
		BaseData: baseData,
	}
	b, err := json.MarshalIndent(item, "", "  ")
	if nil != err {
		t.Fatalf("json error: %s", err)
	}

	t.Logf("BaseData: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *baseData) {
		t.Errorf("different, original: %v  recovered: %v", r, *baseData)
	}
	checkPackedData(t, "base data", packed)
}

// test the pack failure on trying to use the zero public key
func TestPackBaseDataWithZeroAccount(t *testing.T) {

	proofedByAccount := makeAccount(theZeroKey.publicKey)

	r := transactionrecord.OldBaseData{
		Currency:       currency.Bitcoin,
		PaymentAddress: "mipcBbFg9gMiCh81Kj8tqqdgoZub1ZJRfn",
		Owner:          proofedByAccount,
		Nonce:          0x12345678,
		Signature:      []byte{1, 2, 3, 4},
	}

	// test the packer
	_, err := r.Pack(proofedByAccount)
	if nil == err {
		t.Fatalf("pack should have failed")
	}
	if fault.InvalidOwnerOrRegistrant != err {
		t.Fatalf("unexpected pack error: %s", err)
	}
}
