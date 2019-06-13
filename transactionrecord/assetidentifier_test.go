// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
)

// test invalid asset identifiers
func TestInvalidAssetIdentifiers(t *testing.T) {

	invalid := []string{
		"",
		"4b",                         // one byte
		"4bf",                        // odd number of chars
		"4473fb34cc05ed9599935a0098", // truncated
		"4473fb34cc05ed9599935a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de3",    // just one short
		"4473fb34cc05ed9599935a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de379",  // just one char over
		"4473fb34cc05ed9599935a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de3745", // just one byte over

		"BAM04473fb34cc05ed9599935a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de37", // bad prefix
		"ABM04473fb34cc05ed9599935a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de37", // bad prefix
		"QWRT4473fb34cc05ed9599935a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de37", // bad prefix

		"4473fb34cc05ed9599x35a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de37", // invalid hex char x
		"4473fb34cc05ed9599X35a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de37", // invalid hex char X
		"4473fb34cc05ed9599k35a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de37", // invalid hex char k
		"4473fb34cc05ed9599K35a0098ce060dfa546f40932dd7b40d35f8fe5cd6a4ff26f3dbf8ffc86ee8eb6480facfd83f3e20d69bf1e764a59256cf79b89531de37", // invalid hex char K
	}

	for i, textAssetIdentifier := range invalid {
		var link transactionrecord.AssetIdentifier
		n, err := fmt.Sscan(textAssetIdentifier, &link)
		if fault.ErrNotAssetIdentifier != err {
			t.Errorf("%d: testing: %q", i, textAssetIdentifier)
			t.Errorf("%d: expected ErrNotAssetIdentifier but got: %v", i, err)
			return
		}
		if 0 != n {
			t.Errorf("%d: testing: %q", i, textAssetIdentifier)
			t.Errorf("%d: hex to link scanned: %d  expected: 0", i, n)
			return
		}
	}
}

// test asset id conversion
func TestAssetIdentifier(t *testing.T) {

	expectedAssetIdentifier := transactionrecord.AssetIdentifier{
		0x37, 0xde, 0x31, 0x95, 0xb8, 0x79, 0xcf, 0x56,
		0x92, 0xa5, 0x64, 0xe7, 0xf1, 0x9b, 0xd6, 0x20,
		0x3e, 0x3f, 0xd8, 0xcf, 0xfa, 0x80, 0x64, 0xeb,
		0xe8, 0x6e, 0xc8, 0xff, 0xf8, 0xdb, 0xf3, 0x26,
		0xff, 0xa4, 0xd6, 0x5c, 0xfe, 0xf8, 0x35, 0x0d,
		0xb4, 0xd7, 0x2d, 0x93, 0x40, 0x6f, 0x54, 0xfa,
		0x0d, 0x06, 0xce, 0x98, 0x00, 0x5a, 0x93, 0x99,
		0x95, 0xed, 0x05, 0xcc, 0x34, 0xfb, 0x73, 0x44,
	}

	textAssetIdentifier := "37de3195b879cf5692a564e7f19bd6203e3fd8cffa8064ebe86ec8fff8dbf326ffa4d65cfef8350db4d72d93406f54fa0d06ce98005a939995ed05cc34fb7344"

	if expectedAssetIdentifier.String() != textAssetIdentifier {
		t.Errorf("asset id(%%s): %s  expected: %s", expectedAssetIdentifier, textAssetIdentifier)
	}

	if fmt.Sprintf("%v", expectedAssetIdentifier) != textAssetIdentifier {
		t.Errorf("asset id(%%v): %v  expected: %s", expectedAssetIdentifier, textAssetIdentifier)
	}

	if fmt.Sprintf("%#v", expectedAssetIdentifier) != "<asset:"+textAssetIdentifier+">" {
		t.Errorf("asset id(%%#v): %#v  expected: %s", expectedAssetIdentifier, "<asset:"+textAssetIdentifier+">")
	}

	var asset transactionrecord.AssetIdentifier
	n, err := fmt.Sscan("37de3195b879cf5692a564e7f19bd6203e3fd8cffa8064ebe86ec8fff8dbf326ffa4d65cfef8350db4d72d93406f54fa0d06ce98005a939995ed05cc34fb7344", &asset)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}
	if 1 != n {
		t.Fatalf("hex to link scanned: %d  expected: 1", n)
	}

	if asset != expectedAssetIdentifier {
		t.Errorf("asset: %#v  expected: %#v", asset, expectedAssetIdentifier)
		t.Errorf("*** GENERATED asset:\n%s", util.FormatBytes("expectedAssetIdentifier", asset[:]))
	}

	// check JSON conversion
	expectedJSON := `{"AssetIdentifier":"37de3195b879cf5692a564e7f19bd6203e3fd8cffa8064ebe86ec8fff8dbf326ffa4d65cfef8350db4d72d93406f54fa0d06ce98005a939995ed05cc34fb7344"}`

	item := struct {
		AssetIdentifier transactionrecord.AssetIdentifier
	}{
		asset,
	}
	convertedJSON, err := json.Marshal(item)
	if nil != err {
		t.Fatalf("marshal json error: %s", err)
	}
	if expectedJSON != string(convertedJSON) {
		t.Errorf("JSON converted: %q", convertedJSON)
		t.Errorf("     expected:  %q", expectedJSON)
	}

	// test json unmarshal
	var newItem struct {
		AssetIdentifier transactionrecord.AssetIdentifier
	}
	err = json.Unmarshal([]byte(expectedJSON), &newItem)
	if nil != err {
		t.Fatalf("unmarshal json error: %s", err)
	}

	if newItem.AssetIdentifier != expectedAssetIdentifier {
		t.Errorf("link: %#v  expected: %#v", newItem.AssetIdentifier, expectedAssetIdentifier)
	}

}

// test asset id bytes
func TestAssetIdentifierFromBytes(t *testing.T) {

	expectedAssetId := transactionrecord.AssetIdentifier{
		0x37, 0xde, 0x31, 0x95, 0xb8, 0x79, 0xcf, 0x56,
		0x92, 0xa5, 0x64, 0xe7, 0xf1, 0x9b, 0xd6, 0x20,
		0x3e, 0x3f, 0xd8, 0xcf, 0xfa, 0x80, 0x64, 0xeb,
		0xe8, 0x6e, 0xc8, 0xff, 0xf8, 0xdb, 0xf3, 0x26,
		0xff, 0xa4, 0xd6, 0x5c, 0xfe, 0xf8, 0x35, 0x0d,
		0xb4, 0xd7, 0x2d, 0x93, 0x40, 0x6f, 0x54, 0xfa,
		0x0d, 0x06, 0xce, 0x98, 0x00, 0x5a, 0x93, 0x99,
		0x95, 0xed, 0x05, 0xcc, 0x34, 0xfb, 0x73, 0x44,
	}

	valid := []byte{
		0x37, 0xde, 0x31, 0x95, 0xb8, 0x79, 0xcf, 0x56,
		0x92, 0xa5, 0x64, 0xe7, 0xf1, 0x9b, 0xd6, 0x20,
		0x3e, 0x3f, 0xd8, 0xcf, 0xfa, 0x80, 0x64, 0xeb,
		0xe8, 0x6e, 0xc8, 0xff, 0xf8, 0xdb, 0xf3, 0x26,
		0xff, 0xa4, 0xd6, 0x5c, 0xfe, 0xf8, 0x35, 0x0d,
		0xb4, 0xd7, 0x2d, 0x93, 0x40, 0x6f, 0x54, 0xfa,
		0x0d, 0x06, 0xce, 0x98, 0x00, 0x5a, 0x93, 0x99,
		0x95, 0xed, 0x05, 0xcc, 0x34, 0xfb, 0x73, 0x44,
	}

	var assetId transactionrecord.AssetIdentifier
	err := transactionrecord.AssetIdentifierFromBytes(&assetId, valid)
	if nil != err {
		t.Fatalf("AssetIdentifierFromBytes error: %s", err)
	}

	if assetId != expectedAssetId {
		t.Fatalf("assetIdentifier expected: %v  actual: %v", expectedAssetId, assetId)
	}

	err = transactionrecord.AssetIdentifierFromBytes(&assetId, valid[1:])
	if fault.ErrNotAssetIdentifier != err {
		t.Fatalf("AssetIdentifierFromBytes error: %s", err)
	}
}
