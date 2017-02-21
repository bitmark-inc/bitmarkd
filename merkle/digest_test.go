// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package merkle_test

import (
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/util"
	"testing"
)

func TestScanFmt(t *testing.T) {

	// big endian
	stringDigest := "00000000440b921e1b77c6c0487ae5616de67f788f44ae2a5af6e2194d16b6f8"

	var d merkle.Digest
	n, err := fmt.Sscan(stringDigest, &d)
	if nil != err {
		t.Fatalf("hex to digest error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	// bytes as little endian format
	expected := merkle.Digest{
		0xf8, 0xb6, 0x16, 0x4d,
		0x19, 0xe2, 0xf6, 0x5a,
		0x2a, 0xae, 0x44, 0x8f,
		0x78, 0x7f, 0xe6, 0x6d,
		0x61, 0xe5, 0x7a, 0x48,
		0xc0, 0xc6, 0x77, 0x1b,
		0x1e, 0x92, 0x0b, 0x44,
		0x00, 0x00, 0x00, 0x00,
	}

	// show little endian values here
	//if !bytes.Equal(d, expected) {
	if d != expected {
		t.Errorf("digest(LE) = %#v expected %x#v", d, expected)
	}

	s := fmt.Sprintf("%s", d)
	if s != stringDigest {
		t.Errorf("string: digest = %s expected %s", s, stringDigest)
	}

	s = fmt.Sprintf("%#v", d)
	if s != "<SHA3-256-BE:"+stringDigest+">" {
		t.Errorf("hash-v: digest = %s expected %s", s, stringDigest)
	}
}

func TestDigest(t *testing.T) {
	s := []byte("hello world")
	d := merkle.NewDigest(s)

	// big endian
	// printf '%s' 'hello world' | sha3sum -a 256 | awk '{for(i=length($1);i>0;i-=2)x=x substr($1,i-1,2);print x}'
	stringDigest := "38394ef2fb3b1ca394fd72d9a1fb71caf322769ec8aa9909047343567ecc4b64"

	var expected merkle.Digest
	n, err := fmt.Sscan(stringDigest, &expected)
	if nil != err {
		t.Fatalf("hex to digest error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if d != expected {
		t.Errorf("digest = %#v expected %#v", d, expected)
	}
}

// originally part of transaction testing
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
		var link merkle.Digest
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

// originally part of transaction testing
func TestLink(t *testing.T) {

	expectedLink := merkle.Digest{
		0xaa, 0x16, 0xda, 0xcf, 0xba, 0x34, 0x08, 0x29,
		0xc8, 0x57, 0x97, 0xb7, 0xb6, 0xa7, 0x88, 0x72,
		0xc8, 0xc7, 0xec, 0x48, 0x4b, 0xe1, 0x97, 0xc0,
		0xad, 0x2e, 0xa3, 0xa2, 0x1c, 0x13, 0xf8, 0x4b,
	}

	textLink := "4bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa"

	if fmt.Sprintf("%s", expectedLink) != textLink {
		t.Errorf("link(%%s): %s  expected: %s", expectedLink, textLink)
	}

	if fmt.Sprintf("%v", expectedLink) != textLink {
		t.Errorf("link(%%v): %v  expected: %s", expectedLink, textLink)
	}

	if fmt.Sprintf("%#v", expectedLink) != "<SHA3-256-BE:"+textLink+">" {
		t.Errorf("link(%%#v): %#v  expected: %#v", expectedLink, expectedLink)
	}

	var link merkle.Digest
	n, err := fmt.Sscan("4bf8131ca2a32eadc097e14b48ecc7c87288a7b6b79757c8290834bacfda16aa", &link)
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
		t.Errorf("*** GENERATED link:\n%s", util.FormatBytes("expectedLink", link[:]))
	}

	// check JSON conversion
	expectedJSON := `{"Link":"aa16dacfba340829c85797b7b6a78872c8c7ec484be197c0ad2ea3a21c13f84b"}`

	item := struct {
		Link merkle.Digest
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
		Link merkle.Digest
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

// originally part of transaction testing
func TestLinkFromBytes(t *testing.T) {

	expectedLink := merkle.Digest{
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

	var link merkle.Digest
	err := merkle.DigestFromBytes(&link, valid)
	if nil != err {
		t.Fatalf("LinkFromBytes error: %v", err)
	}

	if link != expectedLink {
		t.Fatalf("link expected: %v  actual: %v", expectedLink, link)
	}

	err = merkle.DigestFromBytes(&link, valid[1:])
	if fault.ErrNotLink != err {
		t.Fatalf("LinkFromBytes error: %v", err)
	}
}
