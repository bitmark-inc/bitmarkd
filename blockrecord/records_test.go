// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"testing"
)

type recordsTestType struct {
	leVersion          string
	leTransactionCount string
	leNumber           string
	lePrevious         string
	leMerkle           string
	leTimestamp        string
	leDifficultyBits   string
	leNonce            string
	beExpectedDigest   string
}

var recordsTestData = recordsTestType{
	leVersion:          "0100",
	leTransactionCount: "0100",
	leNumber:           "2000000000000000",
	lePrevious:         "81cd02ab7e569e8bcd9317e2fe99f2de44d49ab2b8851ba4a308000000000000",
	leMerkle:           "e320b6c2fffc8d750423db8b1eb942ae710e951ed797f7affc8892b0f1fc122b",
	leTimestamp:        "c7f5d74d00000000",
	leDifficultyBits:   "f2b9441a3243250d",
	leNonce:            "42a1469535a7d421",

	beExpectedDigest: "08c7539a6d2cf618637f3db6792af495273b04ea946dfc170e6ca4b71fbf1d46",
}

func TestBlockDigestFromHex(t *testing.T) {
	r := recordsTestData // the test data block

	leBlock := r.leVersion + r.leTransactionCount + r.leNumber + r.lePrevious + r.leMerkle + r.leTimestamp + r.leDifficultyBits + r.leNonce

	leBinaryBlock, err := hex.DecodeString(leBlock)

	d := blockdigest.NewDigest(leBinaryBlock)

	var expected blockdigest.Digest
	n, err := fmt.Sscan(r.beExpectedDigest, &expected)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if d != expected {
		t.Logf("block: %x", leBinaryBlock)
		t.Errorf("digest: %#v  expected: %#v", d, expected)
	}
}

func blockDigestFromLittleEndian(t *testing.T, s string) *blockdigest.Digest {

	d := &blockdigest.Digest{}

	_, err := fmt.Sscan(s, d)
	if nil != err {
		t.Fatalf("hex(%s) to block digest error: %v", s, err)
	}

	// need to reverse
	l := len(d)
	for i := 0; i < l/2; i += 1 {
		d[i], d[l-i-1] = d[l-i-1], d[i]
	}

	return d
}

func merkleDigestFromLittleEndian(t *testing.T, s string) *merkle.Digest {

	d := &merkle.Digest{}

	_, err := fmt.Sscan(s, d)
	if nil != err {
		t.Fatalf("hex(%s) to merkle digest error: %v", s, err)
	}

	// need to reverse
	l := len(d)
	for i := 0; i < l/2; i += 1 {
		d[i], d[l-i-1] = d[l-i-1], d[i]
	}

	return d
}

func TestBlockDigestFromBlock(t *testing.T) {

	r := recordsTestData // the test data block

	prevLink := blockDigestFromLittleEndian(t, r.lePrevious)
	merkleRoot := merkleDigestFromLittleEndian(t, r.leMerkle)

	fromLE := func(s string) uint64 {
		h, err := hex.DecodeString(s)
		if nil != err {
			t.Fatalf("fromLE hex error: %v", err)
		}
		b := bytes.NewBuffer(h)
		n := len(h)
		switch n {
		case 2:
			n := uint16(0)
			err = binary.Read(b, binary.LittleEndian, &n)
			if nil != err {
				t.Fatalf("fromLE read error: %v", err)
			}
			return uint64(n)
		case 4:
			n := uint32(0)
			err = binary.Read(b, binary.LittleEndian, &n)
			if nil != err {
				t.Fatalf("fromLE read error: %v", err)
			}
			return uint64(n)
		case 8:
			n := uint64(0)
			err = binary.Read(b, binary.LittleEndian, &n)
			if nil != err {
				t.Fatalf("fromLE read error: %v", err)
			}
			return n
		default:
			t.Fatalf("fromLE n can only be 4 or 8 byte hex, not: %d byte", n)
		}
		return 0
	}

	difficulty := difficulty.New()
	difficulty.SetBits(fromLE(r.leDifficultyBits))
	h := blockrecord.Header{
		Version:          uint16(fromLE(r.leVersion)),
		TransactionCount: uint16(fromLE(r.leTransactionCount)),
		Number:           fromLE(r.leNumber),
		PreviousBlock:    *prevLink,
		MerkleRoot:       *merkleRoot,
		Timestamp:        fromLE(r.leTimestamp),
		Difficulty:       difficulty,
		Nonce:            blockrecord.NonceType(fromLE(r.leNonce)),
	}

	p := h.Pack()

	d := blockdigest.NewDigest(p)

	var expected blockdigest.Digest
	n, err := fmt.Sscan(r.beExpectedDigest, &expected)
	if nil != err {
		t.Fatalf("hex to link error: %v", err)
	}

	if 1 != n {
		t.Fatalf("scanned %d items expected to scan 1", n)
	}

	if d != expected {
		t.Logf("packed: %x", p)
		t.Errorf("digest: %#v  expected: %#v", d, expected)
	}

	// check that the method also returns the same result
	dp := p.Digest()

	if dp != expected {
		t.Logf("packed: %x", p)
		t.Errorf("digest: %#v  expected: %#v", dp, expected)
	}
}

// // data taken from recent (at the time of writing) block
// // ***** FIX THIS: this need new values
// func TestRawBlock328656(t *testing.T) {

// 	// block data
// 	expectedHash := "0000000000000000001436c6d5f9118a6a2f00087629dbf6fac3e5cd9672f0b6"
// 	prevBlock := "0000000000000000009cc28fdf5919a73d4e04e6048d1063ef7cdd24dfab49d3"
// 	merkleRoot := "2b44fc83c84e21817b0da633af7733a4872c2415a21bf9f6b4883a5751c3e020"

// 	difficultyBits := difficulty.New()
// 	difficultyBits.SetBits(404472624)
// 	h := block.Header{
// 		Version:       2,
// 		PreviousBlock: hexEndianDigest(prevBlock),
// 		MerkleRoot:    hexEndianDigest(merkleRoot),
// 		Time:          1415178957,
// 		DifficultyBits:          *difficultyBits,
// 		Nonce:         698985022,
// 	}

// 	// pack the block
// 	p := h.Pack()

// 	// get digest and check
// 	dp := p.Digest()
// 	expected := hexEndianDigest(expectedHash)

// 	if dp != expected {
// 		t.Logf("packed: %x", p)
// 		t.Errorf("digest = %#v expected %#v", dp, expected)
// 	}
// }
