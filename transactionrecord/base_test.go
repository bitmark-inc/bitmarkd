// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"bytes"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"golang.org/x/crypto/ed25519"
	"reflect"
	"testing"
)

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
	r.Signature = signature
	//t.Logf("signature: %#v", r.Signature)
	l := util.ToVarint64(uint64(len(signature)))
	expected = append(expected, l...)
	expected = append(expected, signature...)

	// test the packer
	packed, err := r.Pack(proofedbyAccount)
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
	unpacked, n, err := packed.Unpack()
	if nil != err {
		t.Fatalf("unpack error: %s", err)
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
		t.Fatalf("json error: %s", err)
	}

	t.Logf("BaseData: JSON: %s", b)

	// check that structure is preserved through Pack/Unpack
	// note reg is a pointer here
	if !reflect.DeepEqual(r, *baseData) {
		t.Errorf("different, original: %v  recovered: %v", r, *baseData)
	}
}
