// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"testing"
)

type recordsTestType struct {
	leVersion        string
	lePrevious       string
	leMerkle         string
	leTime           string
	leBits           string
	leNonce          string
	beExpectedDigest string
}

var recordsTestData = recordsTestType{
	// example taken from: https://en.bitcoin.it/wiki/Block_hashing_algorithm
	// these are the actual hex values in little endian form
	leVersion:  "01000000",
	lePrevious: "81cd02ab7e569e8bcd9317e2fe99f2de44d49ab2b8851ba4a308000000000000",
	leMerkle:   "e320b6c2fffc8d750423db8b1eb942ae710e951ed797f7affc8892b0f1fc122b",
	leTime:     "c7f5d74d",
	leBits:     "f2b9441a",
	leNonce:    "42a14695",

	beExpectedDigest: "00000000000000001e8d6829a8a21adc5d38d0a473b144b6765798e61f98bd1d",
}

func TestBlockDigestFromHex(t *testing.T) {
	r := recordsTestData // the test data block

	leBlock := r.leVersion + r.lePrevious + r.leMerkle + r.leTime + r.leBits + r.leNonce

	leBinaryBlock, err := hex.DecodeString(leBlock)

	d := block.NewDigest(leBinaryBlock)

	var expected block.Digest
	n, err := fmt.Sscan(r.beExpectedDigest, &expected)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if d != expected {
		t.Logf("block = %x", leBinaryBlock)
		t.Errorf("digest = %#v expected %#v", d, expected)
	}
}

func digestFromLittleEndian(s string) (*block.Digest, error) {
	var d block.Digest
	_, err := fmt.Sscan(s, &d)
	if nil != err {
		return nil, err
	}

	// need to reverse
	l := len(d)
	for i := 0; i < l/2; i += 1 {
		d[i], d[l-i-1] = d[l-i-1], d[i]
	}
	return &d, nil
}

func TestBlockDigestFromBlock(t *testing.T) {

	r := recordsTestData // the test data block

	prevLink, err := digestFromLittleEndian(r.lePrevious)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	merkle, err := digestFromLittleEndian(r.leMerkle)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	fromLE := func(s string) uint32 {
		h, err := hex.DecodeString(s)
		if nil != err {
			t.Fatalf("fromLE hex error: %v", err)
		}
		b := bytes.NewBuffer(h)
		n := uint32(0)
		err = binary.Read(b, binary.LittleEndian, &n)
		if nil != err {
			t.Fatalf("fromLE read error: %v", err)
		}
		return n
	}

	bits := difficulty.New()
	bits.SetUint32(fromLE(r.leBits))
	h := block.Header{
		Version:       fromLE(r.leVersion),
		PreviousBlock: *prevLink,
		MerkleRoot:    *merkle,
		Time:          fromLE(r.leTime),
		Bits:          *bits,
		Nonce:         fromLE(r.leNonce),
	}

	p := h.Pack()

	d := block.NewDigest(p)

	var expected block.Digest
	n, err := fmt.Sscan(r.beExpectedDigest, &expected)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if d != expected {
		t.Logf("packed: %x", p)
		t.Errorf("digest = %#v expected %#v", d, expected)
	}

	// check that the method also returns the same result
	dp := p.Digest()

	if dp != expected {
		t.Logf("packed: %x", p)
		t.Errorf("digest = %#v expected %#v", dp, expected)
	}

}

// data taken from recent (at the time of writing) block
func TestRawBlock328656(t *testing.T) {

	hexEndianDigest := func(s string) block.Digest {
		var d block.Digest
		n, err := fmt.Sscan(s, &d)
		if nil != err {
			t.Fatalf("hex to link error: %v", err)
		}
		if 1 != n {
			t.Fatalf("scanned %d items expected to scan 1", n)
		}
		return d
	}

	// block data
	expectedHash := "0000000000000000001436c6d5f9118a6a2f00087629dbf6fac3e5cd9672f0b6"
	prevBlock := "0000000000000000009cc28fdf5919a73d4e04e6048d1063ef7cdd24dfab49d3"
	merkleRoot := "2b44fc83c84e21817b0da633af7733a4872c2415a21bf9f6b4883a5751c3e020"

	bits := difficulty.New()
	bits.SetUint32(404472624)
	h := block.Header{
		Version:       2,
		PreviousBlock: hexEndianDigest(prevBlock),
		MerkleRoot:    hexEndianDigest(merkleRoot),
		Time:          1415178957,
		Bits:          *bits,
		Nonce:         698985022,
	}

	// pack the block
	p := h.Pack()

	// get digest and check
	dp := p.Digest()
	expected := hexEndianDigest(expectedHash)

	if dp != expected {
		t.Logf("packed: %x", p)
		t.Errorf("digest = %#v expected %#v", dp, expected)
	}
}
