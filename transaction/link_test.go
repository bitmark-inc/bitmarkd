// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction_test

import (
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"testing"
)

// test invalid links
func TestInvalidLinks(t *testing.T) {

	invalid := []string{
		"",
		"B",
		"BM",
		"BMK",
		"BMK0",

		"BMK04b",                                                                 // one byte
		"BMK04bf",                                                                // odd number of chars
		"BMK04bf8131ca2a32eadc097e14b48",                                         // truncated
		"BMK04bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16a",    // just one char short
		"BMK04bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa6",  // just one char over
		"BMK04bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa68", // just one byte over

		"BKM04bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // bad prefix
		"MKB04bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // bad prefix
		"QWRT4bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // bad prefix
		"BMK04bf8131ca2a32eadc0x7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char x
		"BMK04bf8131ca2a32eadc0X7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char X
		"BMK04bf8131ca2a32eadc0k7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char k
		"BMK04bf8131ca2a32eadc0K7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char K
	}

	for i, textLink := range invalid {
		var link transaction.Link
		n, err := fmt.Sscan(textLink, &link)
		if fault.ErrNotLink != err {
			t.Errorf("%d: testing: %q", i, textLink)
			t.Errorf("%d: expected ErrNotLink but got: %v", i, err)
			return
		}
		if 0 != n {
			t.Errorf("%d: testing: %q", i, textLink)
			t.Errorf("%d: hex to link scanned: %d  expected: 0", i, n)
			return
		}
	}
}

// test link conversion
func TestLink(t *testing.T) {

	expectedLink := transaction.Link{
		0xaa, 0x16, 0xda, 0xcf, 0xba, 0x34, 0x08, 0x29,
		0xc8, 0x57, 0x97, 0xb7, 0xb6, 0xa7, 0x88, 0x72,
		0xc8, 0xc7, 0xec, 0x48, 0x4b, 0xe1, 0x97, 0xc0,
		0xad, 0x2e, 0xa3, 0xa2, 0x1c, 0x13, 0xf8, 0x4b,
	}

	textLink := "4bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa"

	if fmt.Sprintf("%s", expectedLink) != textLink {
		t.Errorf("link(%%s): %s  expected: %x", expectedLink, textLink)
	}

	if fmt.Sprintf("%v", expectedLink) != textLink {
		t.Errorf("link(%%v): %v  expected: %x", expectedLink, textLink)
	}

	if fmt.Sprintf("%#v", expectedLink) != "<link:"+textLink+">" {
		t.Errorf("link(%%#v): %#v  expected: %#v", expectedLink, expectedLink)
	}

	var link transaction.Link
	n, err := fmt.Sscan("BMK04bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", &link)
	if nil != err {
		t.Errorf("hex to link error: %v", err)
		return
	}
	if 1 != n {
		t.Errorf("hex to link scanned: %d  expected: 1", n)
		return
	}

	if link != expectedLink {
		t.Errorf("link: %#v  expected: %#v", link, expectedLink)
		t.Errorf("*** GENERATED link:\n%s", formatBytes("expectedLink", link.Bytes()))
	}

	// check JSON conversion
	expectedJSON := `{"Link":"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b"}`

	item := struct {
		Link transaction.Link
	}{
		link,
	}
	convertedJSON, err := json.Marshal(item)
	if nil != err {
		t.Errorf("marshal json error: %v", err)
		return
	}
	if expectedJSON != string(convertedJSON) {
		t.Errorf("JSON converted: %q", convertedJSON)
		t.Errorf("     expected:  %q", expectedJSON)
	}

	// test json unmarshal
	var newItem struct {
		Link transaction.Link
	}
	err = json.Unmarshal([]byte(expectedJSON), &newItem)
	if nil != err {
		t.Errorf("unmarshal json error: %v", err)
		return
	}

	if newItem.Link != expectedLink {
		t.Errorf("link: %#v  expected: %#v", newItem.Link, expectedLink)
	}

}

// test hex link conversion
func TestHexLink(t *testing.T) {

	// little endian
	expectedLink := transaction.Link{
		0xaa, 0x16, 0xda, 0xcf, 0xba, 0x34, 0x08, 0x29,
		0xc8, 0x57, 0x97, 0xb7, 0xb6, 0xa7, 0x88, 0x72,
		0xc8, 0xc7, 0xec, 0x48, 0x4b, 0xe1, 0x97, 0xc0,
		0xad, 0x2e, 0xa3, 0xa2, 0x1c, 0x13, 0xf8, 0x4b,
	}

	// little endian
	prefixedHex := "424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b"

	var link transaction.Link
	err := transaction.LinkFromHexString(&link, prefixedHex)
	if nil != err {
		t.Errorf("LinkFromHexString error: %v", err)
		return
	}

	if link != expectedLink {
		t.Errorf("link: %#v  expected: %#v", link, expectedLink)
	}

}

// test invalid hex link conversion
func TestInvalidHexLink(t *testing.T) {

	// little endian
	expectedLink := transaction.Link{}

	// bad values
	testValues := []string{
		"404d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f8",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c1",
		"434d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b",
		"424d4b30aa",
		"424d4b30a",
		"424d4b30",
		"424d4b3",
		"424d4b",
		"424d4",
		"424d",
		"424",
		"42",
		"4",
		"",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b8435983405932094750927692406802486092808420986042832468365",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b843598340593209475092769240680248609280842098604283246836",
		"424d4b30aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b84359834059320947509276924068024860928084209860428324683",
	}

	for i, prefixedHex := range testValues {
		var link transaction.Link
		err := transaction.LinkFromHexString(&link, prefixedHex)
		if nil == err {
			t.Errorf("%d: LinkFromHexString unexpected success", i)
		}

		if link != expectedLink {
			t.Errorf("%d: link: %#v  expected: %#v", i, link, expectedLink)
		}
	}
}
