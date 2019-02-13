// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockdigest_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
)

func TestScanFmt(t *testing.T) {

	// big endian
	stringDigest := "00000000440b921e1b77c6c0487ae5616de67f788f44ae2a5af6e2194d16b6f8"

	var d blockdigest.Digest
	n, err := fmt.Sscan(stringDigest, &d)
	if nil != err {
		t.Fatalf("hex to digest error: %s", err)
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
		t.Fatalf("hex to big error: %s", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if 0 != d.Cmp(&expectedBig) {
		t.Errorf("digest: %s != expected: %s", d, expectedBig.Text(16))
	}

}

func TestDigest(t *testing.T) {
	s := []byte("hello world")
	d := blockdigest.NewDigest(s)

	// big endian
	// printf '%s' 'hello world' | argon2 'hello world' -d -l 32 -m 18 -t 5 -p 1 -r | awk '{for(i=length($1);i>0;i-=2)x=x substr($1,i-1,2);print x}'
	stringDigest := "f8a17bc25cb53e848e2d09811ade4b8a037f628443661b88611faf5d9a5a1f33"

	var expected blockdigest.Digest
	n, err := fmt.Sscan(stringDigest, &expected)
	if nil != err {
		t.Fatalf("hex to digest error: %s", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if d != expected {
		t.Errorf("digest = %#v expected %#v", d, expected)
	}
}

func TestBlockDataDigest(t *testing.T) {

	blockdata := []byte{
		0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x1e, 0x9a, 0xcb, 0x62,
		0x37, 0xc5, 0x6b, 0x38, 0x91, 0x26, 0x40, 0x27,
		0x2d, 0x74, 0xd1, 0xb4, 0x70, 0xb8, 0xfd, 0x22,
		0xa6, 0x9a, 0x04, 0xeb, 0x6b, 0xf8, 0x72, 0xb5,
		0xd1, 0x01, 0xd5, 0x26, 0xb7, 0x9a, 0x80, 0x56,
		0xff, 0xff, 0x00, 0x1d, 0xa0, 0x00, 0x28, 0xd4,
	}

	d := blockdigest.NewDigest(blockdata)

	stringDigest := "799caa04b138ebf218a37bc63a0ceadc9c3274402618b5e369725191c0c5fa6e"
	expectedJSON := `"6efac5c091517269e3b518264074329cdcea0c3ac67ba318f2eb38b104aa9c79"`

	var expected blockdigest.Digest
	n, err := fmt.Sscan(stringDigest, &expected)
	if nil != err {
		t.Fatalf("hex to digest error: %s", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if expected != d {
		t.Errorf("digest: expected: %#v actual: %#v", expected, d)
	}

	// test JSON
	buffer, err := json.Marshal(d)
	if nil != err {
		t.Fatalf("marshal JSON error: %s", err)
	}

	if expectedJSON != string(buffer) {
		t.Errorf("json digest expected: %s  actual: %s", expectedJSON, buffer)
	}

	var jd blockdigest.Digest
	err = json.Unmarshal([]byte(expectedJSON), &jd)
	if nil != err {
		t.Fatalf("unmarshal JSON error: %s", err)
	}

	if d != jd {
		t.Errorf("digest: expected: %#v  actual: %#v", d, jd)
	}
}
