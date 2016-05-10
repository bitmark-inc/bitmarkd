// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdigest_test

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"math/big"
	"testing"
)

func TestScanFmt(t *testing.T) {

	// big endian
	stringDigest := "00000000440b921e1b77c6c0487ae5616de67f788f44ae2a5af6e2194d16b6f8"

	var d blockdigest.Digest
	n, err := fmt.Sscan(stringDigest, &d)
	if nil != err {
		t.Fatalf("hex to digest error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	// bytes as little endian format
	expected := blockdigest.Digest{
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
	if s != "<Argon2d:"+stringDigest+">" {
		t.Errorf("hash-v: digest = %s expected %s", s, stringDigest)
	}

	var expectedBig big.Int
	n, err = fmt.Sscanf(stringDigest, "%x", &expectedBig)
	if nil != err {
		t.Fatalf("hex to big error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if 0 != d.Cmp(&expectedBig) {
		t.Errorf("digest: %s != expected: %x", d, expectedBig)
	}

}

func TestDigest(t *testing.T) {
	s := []byte("hello world")
	d := blockdigest.NewDigest(s)

	// big endian
	// printf '%s' 'hello world' | argon2 'BitmarkBlockchain2016' -d -h 32 -m 18 -t 5 -p 1 -r | awk '{for(i=length($1);i>0;i-=2)x=x substr($1,i-1,2);print x}'
	stringDigest := "6e3c507fabee4eabc74a79d27a7f69250978c4cb468b8c4a2b457bb766049ea6"

	var expected blockdigest.Digest
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
