// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord_test

import (
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"testing"
)

// test invalid asset indexes
func TestInvalidAssetIndexes(t *testing.T) {

	invalid := []string{
		"",
		"4b",  // one byte
		"4bf", // odd number of chars
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

	for i, textAssetIndex := range invalid {
		var link transactionrecord.AssetIndex
		n, err := fmt.Sscan(textAssetIndex, &link)
		if fault.ErrNotAssetIndex != err {
			t.Errorf("%d: testing: %q", i, textAssetIndex)
			t.Errorf("%d: expected ErrNotAssetIndex but got: %v", i, err)
			return
		}
		if 0 != n {
			t.Errorf("%d: testing: %q", i, textAssetIndex)
			t.Errorf("%d: hex to link scanned: %d  expected: 0", i, n)
			return
		}
	}
}

// test asset index conversion
func TestAssetIndex(t *testing.T) {

	expectedAssetIndex := transactionrecord.AssetIndex{
		0x37, 0xde, 0x31, 0x95, 0xb8, 0x79, 0xcf, 0x56,
		0x92, 0xa5, 0x64, 0xe7, 0xf1, 0x9b, 0xd6, 0x20,
		0x3e, 0x3f, 0xd8, 0xcf, 0xfa, 0x80, 0x64, 0xeb,
		0xe8, 0x6e, 0xc8, 0xff, 0xf8, 0xdb, 0xf3, 0x26,
		0xff, 0xa4, 0xd6, 0x5c, 0xfe, 0xf8, 0x35, 0x0d,
		0xb4, 0xd7, 0x2d, 0x93, 0x40, 0x6f, 0x54, 0xfa,
		0x0d, 0x06, 0xce, 0x98, 0x00, 0x5a, 0x93, 0x99,
		0x95, 0xed, 0x05, 0xcc, 0x34, 0xfb, 0x73, 0x44,
	}

	textAssetIndex := "37de3195b879cf5692a564e7f19bd6203e3fd8cffa8064ebe86ec8fff8dbf326ffa4d65cfef8350db4d72d93406f54fa0d06ce98005a939995ed05cc34fb7344"

	if fmt.Sprintf("%s", expectedAssetIndex) != textAssetIndex {
		t.Errorf("asset index(%%s): %s  expected: %s", expectedAssetIndex, textAssetIndex)
	}

	if fmt.Sprintf("%v", expectedAssetIndex) != textAssetIndex {
		t.Errorf("asset index(%%v): %v  expected: %s", expectedAssetIndex, textAssetIndex)
	}

	if fmt.Sprintf("%#v", expectedAssetIndex) != "<asset:"+textAssetIndex+">" {
		t.Errorf("asset index(%%#v): %#v  expected: %s", expectedAssetIndex, "<asset:"+textAssetIndex+">")
	}

	var asset transactionrecord.AssetIndex
	n, err := fmt.Sscan("37de3195b879cf5692a564e7f19bd6203e3fd8cffa8064ebe86ec8fff8dbf326ffa4d65cfef8350db4d72d93406f54fa0d06ce98005a939995ed05cc34fb7344", &asset)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
	}
	if 1 != n {
		t.Fatalf("hex to link scanned: %d  expected: 1", n)
	}

	if asset != expectedAssetIndex {
		t.Errorf("asset: %#v  expected: %#v", asset, expectedAssetIndex)
		t.Errorf("*** GENERATED asset:\n%s", util.FormatBytes("expectedAssetIndex", asset[:]))
	}

	// check JSON conversion
	expectedJSON := `{"AssetIndex":"37de3195b879cf5692a564e7f19bd6203e3fd8cffa8064ebe86ec8fff8dbf326ffa4d65cfef8350db4d72d93406f54fa0d06ce98005a939995ed05cc34fb7344"}`

	item := struct {
		AssetIndex transactionrecord.AssetIndex
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
		AssetIndex transactionrecord.AssetIndex
	}
	err = json.Unmarshal([]byte(expectedJSON), &newItem)
	if nil != err {
		t.Fatalf("unmarshal json error: %s", err)
	}

	if newItem.AssetIndex != expectedAssetIndex {
		t.Errorf("link: %#v  expected: %#v", newItem.AssetIndex, expectedAssetIndex)
	}

}

// test asset index bytes
func TestAssetIndexFromBytes(t *testing.T) {

	expectedAssetIndex := transactionrecord.AssetIndex{
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

	var assetIndex transactionrecord.AssetIndex
	err := transactionrecord.AssetIndexFromBytes(&assetIndex, valid)
	if nil != err {
		t.Fatalf("AssetIndexFromBytes error: %s", err)
	}

	if assetIndex != expectedAssetIndex {
		t.Fatalf("assetIndex expected: %v  actual: %v", expectedAssetIndex, assetIndex)
	}

	err = transactionrecord.AssetIndexFromBytes(&assetIndex, valid[1:])
	if fault.ErrNotAssetIndex != err {
		t.Fatalf("AssetIndexFromBytes error: %s", err)
	}
}
