// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mine_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	//"github.com/bitmark-inc/bitmarkd/mine"
	"testing"
)

// set this true to get logging ouptuts from various tests
//const verboseTesting = true
const verboseTesting = false

// sample data sent to miner
var allTestData = struct {
	previousBlock  string
	expectedDigest string

	coinb1      string
	coinb2      string
	extraNonce1 string
	extraNonce2 string

	version string
	time    string
	bits    string
	nonce   string
}{
	// example taken from: https://en.bitcoin.it/wiki/Block_hashing_algorithm
	// see the block at: http://blockexplorer.com/testnet/b/25096

	previousBlock:  "00000000440b921e1b77c6c0487ae5616de67f788f44ae2a5af6e2194d16b6f8",
	expectedDigest: "0000000042a81be20b4bde68cdf0878c32b9a6304218907129ef2bd4a7cc28b0",

	coinb1:      "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff20020862062f503253482f04b8864e5008",
	coinb2:      "072f736c7573682f000000000100f2052a010000001976a914d23fcdf86f7e756a64a7a9688ef9903327048ed988ac00000000",
	extraNonce1: "08000002",
	extraNonce2: "00000000",

	version: "00000002",
	time:    "504e86b9",
	bits:    "1c2ac4af",
	nonce:   "7e92eda0", // miner calculated this
}

// data captured from debugging running the miner with the above data
var allCapturedData = struct {
	coinbase       string
	beMerkleRoot   string
	block          string
	expectedDigest string
}{
	coinbase: "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff20020862062f503253482f04b8864e50080800000200000000072f736c7573682f000000000100f2052a010000001976a914d23fcdf86f7e756a64a7a9688ef9903327048ed988ac00000000",

	beMerkleRoot: "2ddeda34d3c88cd28d7a858d4c9846558fc2a9c96bb57e48a244cf2ed17ec42b",

	// each 32 bit word needs to be individually byte swapped
	block: "000000024d16b6f85af6e2198f44ae2a6de67f78487ae5611b77c6c0440b921e00000000d17ec42ba244cf2e6bb57e488fc2a9c94c9846558d7a858dd3c88cd22ddeda34504e86b91c2ac4af7e92eda0",
	// padding removed from block: "000000800000000000000000000000000000000000000000000000000000000000000000000000000000000080020000"
}

// coinbase data captured from the miner
func TestCapturedCoinbase(t *testing.T) {

	c := allCapturedData // debug data from miner

	d := block.NewDigest(hexToBinary(t, c.coinbase))

	expectedMerkle := digestFromHex(t, c.beMerkleRoot)

	if d != expectedMerkle {
		t.Logf("digest = %#v", d)
		t.Errorf("got %#v  expected: %#v", d, expectedMerkle)
	}
}

// block data captured from the miner
func TestCapturedBlock(t *testing.T) {

	r := allTestData     // the test data block
	c := allCapturedData // debug data from miner

	b1 := hexToBinary(t, c.block)
	d1 := block.NewDigest(b1) // this will be wrong!

	b2 := make([]byte, len(b1))

	// swap 32 bit words to BE form
	// since the dump routine is swapping them for display
	for i := 0; i < len(b1); i += 4 {
		b2[i+3] = b1[i+0]
		b2[i+2] = b1[i+1]
		b2[i+1] = b1[i+2]
		b2[i+0] = b1[i+3]
	}

	d2 := block.NewDigest(b2)

	if 80 != len(b1) || 80 != len(b2) {
		t.Errorf("len(b1) = %d", len(b1))
		t.Errorf("len(b2) = %d", len(b2))
	}

	e := digestFromHex(t, r.expectedDigest)
	if d2 != e {

		t.Logf("b2 = %x", b2)

		t.Logf("digest1 = %#v", d1)
		t.Errorf("digest2 = %#v  expected: %#v", d2, e)
	}
}

// extract the values from captured data and make a block
func TestBlockAssemblyFromCapturedData(t *testing.T) {

	r := allTestData     // the test data block
	c := allCapturedData // debug data from miner

	bits := difficulty.New()
	bits.SetBits(0x1c2ac4af)

	h := block.Header{
		Version:       2,
		PreviousBlock: digestFromHex(t, r.previousBlock),
		MerkleRoot:    digestFromHex(t, c.beMerkleRoot),
		Time:          0x504e86b9,
		Bits:          *bits,
		Nonce:         0x7e92eda0,
	}

	// pack the block
	p := h.Pack()

	// get digest and check
	dp := p.Digest()
	expected := digestFromHex(t, r.expectedDigest)

	if dp != expected {
		t.Logf("packed: %x", p)
		t.Errorf("digest = %#v expected %#v", dp, expected)
	}
}

// test building a block from all the individual bits
func TestBlock(t *testing.T) {

	r := allTestData     // the test data block
	c := allCapturedData // debug data from miner

	coinb1 := hexToBinary(t, r.coinb1)
	coinb2 := hexToBinary(t, r.coinb2)
	extraNonce1 := hexToBinary(t, r.extraNonce1)
	extraNonce2 := hexToBinary(t, r.extraNonce2)

	coinbase := append(coinb1, extraNonce1...)
	coinbase = append(coinbase, extraNonce2...)
	coinbase = append(coinbase, coinb2...)

	cDigest := block.NewDigest(coinbase)

	expectedMerkle := digestFromHex(t, c.beMerkleRoot)

	// --------------------

	// regenerate merkle root
	merkleRoot := cDigest

	if merkleRoot != expectedMerkle {
		t.Logf("coinbase = %x", coinbase)
		t.Logf("merkle root = %#v\n", merkleRoot)
		t.Errorf("merkle root got: %#v  expected %#v", merkleRoot, expectedMerkle)
	}

	bits := difficulty.New()
	bits.SetBits(fromBE(r.bits))

	h := block.Header{
		Version:       fromBE(r.version),
		PreviousBlock: digestFromHex(t, r.previousBlock),
		MerkleRoot:    merkleRoot,
		Time:          fromBE(r.time),
		Bits:          *bits,
		Nonce:         fromBE(r.nonce),
	}

	p := h.Pack()

	// get digest and check
	d := p.Digest()
	expected := digestFromHex(t, r.expectedDigest)

	if d != expected {
		t.Logf("packed: %x", p)
		t.Errorf("digest: %#v  expected: %#v", d, expected)
	}
}

// digest from hex functions
func digestFromHex(t *testing.T, s string) block.Digest {
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

// convert numbers
func fromBE(s string) uint32 {
	return fromAny(s, binary.BigEndian)
}
func fromLE(s string) uint32 {
	return fromAny(s, binary.LittleEndian)
}
func fromAny(s string, endian binary.ByteOrder) uint32 {
	h, err := hex.DecodeString(s)
	if nil != err {
		panic(fmt.Sprintf("fromLE hex error: %v", err))
	}
	b := bytes.NewBuffer(h)
	n := uint32(0)
	err = binary.Read(b, endian, &n)
	if nil != err {
		panic(fmt.Sprintf("fromLE read error: %v", err))
	}
	return n
}

// convert hex string
func hexToBinary(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	if nil != err {
		t.Fatalf("hex decode error: %v", err)
	}
	return b
}

// test that endien converter works
func TestEndian(t *testing.T) {

	b := fromBE("12345678")
	be := uint32(0x12345678)

	l := fromLE("12345678")
	le := uint32(0x78563412)

	if b != be {
		t.Errorf("got: %x  expected: %x", b, be)
	}

	if l != le {
		t.Errorf("got: %x  expected: %x", l, le)
	}
}
