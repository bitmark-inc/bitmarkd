// Copyright (c) 2014-2016 Bitmark Inc.
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

// test invalid links
func TestInvalidLinks(t *testing.T) {

	invalid := []string{
		"",
		"4b",  // one byte
		"4bf", // odd number of chars
		"4bf8131ca2a32eadc097e14b48",                                         // truncated
		"4bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16a",    // just one char short
		"4bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa6",  // just one char over
		"4bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa68", // just one byte over

		"4bf8131ca2a32eadc0x7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char x
		"4bf8131ca2a32eadc0X7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char X
		"4bf8131ca2a32eadc0k7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char k
		"4bf8131ca2a32eadc0K7e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", // invalid hex char K
	}

	for i, textLink := range invalid {
		var link transactionrecord.Link
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

	expectedLink := transactionrecord.Link{
		0xaa, 0x16, 0xda, 0xcf, 0xba, 0x34, 0x08, 0x29,
		0xc8, 0x57, 0x97, 0xb7, 0xb6, 0xa7, 0x88, 0x72,
		0xc8, 0xc7, 0xec, 0x48, 0x4b, 0xe1, 0x97, 0xc0,
		0xad, 0x2e, 0xa3, 0xa2, 0x1c, 0x13, 0xf8, 0x4b,
	}

	textLink := "aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b"

	if fmt.Sprintf("%s", expectedLink) != textLink {
		t.Errorf("link(%%s): %s  expected: %x", expectedLink, textLink)
	}

	if fmt.Sprintf("%v", expectedLink) != textLink {
		t.Errorf("link(%%v): %v  expected: %x", expectedLink, textLink)
	}

	if fmt.Sprintf("%#v", expectedLink) != "<link:"+textLink+">" {
		t.Errorf("link(%%#v): %#v  expected: %#v", expectedLink, expectedLink)
	}

	var link transactionrecord.Link
	n, err := fmt.Sscan("aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b", &link)
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
		t.Errorf("*** GENERATED link:\n%s", util.FormatBytes("expectedLink", link.Bytes()))
	}

	// check JSON conversion
	expectedJSON := `{"Link":"aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b"}`

	item := struct {
		Link transactionrecord.Link
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
		Link transactionrecord.Link
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

// test link bytes
func TestLinkFromBytes(t *testing.T) {

	expectedLink := transactionrecord.Link{
		0xaa, 0x16, 0xda, 0xcf, 0xba, 0x34, 0x08, 0x29,
		0xc8, 0x57, 0x97, 0xb7, 0xb6, 0xa7, 0x88, 0x72,
		0xc8, 0xc7, 0xec, 0x48, 0x4b, 0xe1, 0x97, 0xc0,
		0xad, 0x2e, 0xa3, 0xa2, 0x1c, 0x13, 0xf8, 0x4b,
	}

	valid := []byte{
		0xaa, 0x16, 0xda, 0xcf, 0xba, 0x34, 0x08, 0x29,
		0xc8, 0x57, 0x97, 0xb7, 0xb6, 0xa7, 0x88, 0x72,
		0xc8, 0xc7, 0xec, 0x48, 0x4b, 0xe1, 0x97, 0xc0,
		0xad, 0x2e, 0xa3, 0xa2, 0x1c, 0x13, 0xf8, 0x4b,
	}

	var link transactionrecord.Link
	err := transactionrecord.LinkFromBytes(&link, valid)
	if nil != err {
		t.Fatalf("LinkFromBytes error: %v", err)
	}

	if link != expectedLink {
		t.Fatalf("link expected: %v  actual: %v", expectedLink, link)
	}

	err = transactionrecord.LinkFromBytes(&link, valid[1:])
	if fault.ErrNotLink != err {
		t.Fatalf("LinkFromBytes error: %v", err)
	}
}
