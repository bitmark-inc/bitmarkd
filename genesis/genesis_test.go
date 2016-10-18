// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package genesis_test

import (
	"bytes"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"testing"
	"time"
	"unicode"
)

// some constants embedded into the genesis block
const (
	genesisBlockNumber  = 1
	genesisBlockVersion = 1
)

// hold chain specific timestamp
type TS struct {
	timestamp uint64
	utc       string
}

// data block
type SourceData struct {
	Timestamp TS
	Nonce     blockrecord.NonceType
	Nonce2    uint64
	Message   string
	ProofedBy string
	Signature []byte
}

// some data embedded into the genesis block
// for live chain
var LiveNet = SourceData{
	// date -u -r $(printf '%d\n' 0x56809ab7)
	// Mon 28 Dec 2015 02:13:11 UTC
	// date -u -r $(printf '%d\n' 0x56809ab7) '+%FT%TZ'
	// 2015-12-28T02:13:11Z
	Timestamp: TS{0x56809ab7, "2015-12-28T02:13:11Z"},

	Nonce: 0x4fa9713e80a6d2ed,

	Nonce2: 0x4c6976652a4e6574,

	Message: "DOWN the RABBIT hole",

	ProofedBy: "acRQJLJtHH61bfoQydREnvXDQ4Tt2BLmGbP1UbcFpJouJSM5hG",

	Signature: []byte{
		0xc2, 0xa3, 0x84, 0xeb, 0xc9, 0x01, 0xa1, 0x8a,
		0x13, 0xa2, 0x70, 0xaa, 0x9f, 0x5e, 0x08, 0x06,
		0x77, 0xd7, 0xab, 0x2f, 0xd8, 0x88, 0xa5, 0xf6,
		0x57, 0xd2, 0xc6, 0xd4, 0x69, 0x2e, 0x6f, 0xcd,
		0xe7, 0x1c, 0x04, 0xb9, 0x1b, 0xe1, 0x40, 0x0e,
		0x7c, 0x1e, 0x8d, 0x5e, 0x2b, 0x34, 0x83, 0xc4,
		0x77, 0xfe, 0xa1, 0x7b, 0xc1, 0xde, 0xe0, 0x05,
		0xcc, 0x8d, 0x4d, 0xf8, 0x62, 0x77, 0x0d, 0x0c,
	},
}

// some data embedded into the genesis block
// for test chain
var TestNet = SourceData{
	// date -u -r $(printf '%d\n' 0x5478424b)
	// Fri Nov 28 09:37:15 UTC 2014
	// date -u -r $(printf '%d\n' 0x5478424b) '+%FT%TZ'
	// 2014-11-28T09:37:15Z
	Timestamp: TS{0x5478424b, "2014-11-28T09:37:15Z"},

	Nonce: 0x473640eeca2b4cd4,

	Nonce2: 0x546573742a4e6574,

	Message: "Bitmark Testing Genesis Block",

	ProofedBy: "fHrBioy1AMn86jJj1rk5j5rokqQhz8hABmccHjfxp9JkAF1dJz",

	Signature: []byte{
		0x02, 0xa8, 0xbf, 0x5c, 0x21, 0x73, 0x03, 0x24,
		0x04, 0x40, 0x79, 0xa5, 0x78, 0x0a, 0x9c, 0xd2,
		0x2f, 0xc2, 0x22, 0xb4, 0x4c, 0x91, 0x29, 0x17,
		0xce, 0xa5, 0xb9, 0xd3, 0x77, 0x0c, 0x13, 0x8e,
		0x8d, 0x3e, 0xae, 0x98, 0xb7, 0x6c, 0x2e, 0x93,
		0xa9, 0x7e, 0x41, 0xc4, 0x1b, 0xae, 0x36, 0xc8,
		0x41, 0x37, 0x08, 0xa9, 0x94, 0xfe, 0xc2, 0xf9,
		0xeb, 0xc0, 0xf8, 0x02, 0x98, 0x3d, 0xf6, 0x01,
	},
}

// create the live genesis block
//
// must be first
func TestLiveGenesisAssembly(t *testing.T) {
	checkAssembly(t, "Live", LiveNet, genesis.LiveGenesisDigest, genesis.LiveGenesisBlock)
}

// create the test genesis block
//
// must be after the live test
func TestTestGenesisAssembly(t *testing.T) {

	mode.Initialise(chain.Testing) // enter test mode - ONLY ALLOWED ONCE (or panic will occur

	checkAssembly(t, "Test", TestNet, genesis.TestGenesisDigest, genesis.TestGenesisBlock)
}

func checkAssembly(t *testing.T, title string, source SourceData, gDigest blockdigest.Digest, gBlock []byte) {

	proofedbyAccount, err := account.AccountFromBase58(source.ProofedBy)
	if nil != err {
		t.Fatalf("failed to parse account: error: %v", err)
	}

	timestamp, err := time.Parse(time.RFC3339, source.Timestamp.utc)
	if nil != err {
		t.Fatalf("failed to parse time: error: %v", err)
	}
	timeUint64 := uint64(timestamp.UTC().Unix())
	if timeUint64 != source.Timestamp.timestamp {
		t.Fatalf("time converted to: 0x%x  expected: %x", timeUint64, source.Timestamp.timestamp)
	}

	// some common static data
	previousBlock := blockdigest.Digest{}

	b := transactionrecord.BaseData{
		Currency:       currency.Nothing,
		PaymentAddress: source.Message,
		Owner:          proofedbyAccount,
		Nonce:          source.Nonce2,
		Signature:      source.Signature,
	}

	base, err := b.Pack(proofedbyAccount)
	if nil != err {
		t.Fatalf("failed to pack base: error: %v", err)
	}
	baseDigest := merkle.Digest(base.MakeLink())

	// merkle tree
	tree := merkle.FullMerkleTree([]merkle.Digest{baseDigest})
	if tree[len(tree)-1] != baseDigest {
		t.Fatalf("failed to compute tree: actual: %#v  expected: %#v", tree[len(tree)-1], baseDigest)
	}

	// default difficulty
	bits := difficulty.New() // defaults to 1

	// block header
	h := blockrecord.Header{
		Version:          genesisBlockVersion,
		TransactionCount: 1,
		Number:           genesisBlockNumber,
		PreviousBlock:    previousBlock,
		MerkleRoot:       tree[len(tree)-1], // replace with message?
		Timestamp:        source.Timestamp.timestamp,
		Difficulty:       bits,
		Nonce:            source.Nonce,
	}

	header := h.Pack()
	hDigest := header.Digest()

	// ok - log the header and coinbase data
	t.Logf("Title: %s", title)
	t.Logf("header: %#v\n", h)
	t.Logf("packed header: %x\n", header)
	t.Logf("base: %x\n", base)
	t.Logf("base digest: %#v\n", baseDigest)
	t.Logf("merkle tree: %#v\n", tree)
	t.Logf("merkle root little endian hex: %x\n", [blockdigest.Length]byte(tree[0]))
	t.Logf("hDigest: %#v\n", hDigest)
	t.Logf("hDigest little endian hex: %x\n", [blockdigest.Length]byte(hDigest))

	// check that it matches
	if hDigest != gDigest {
		t.Errorf("digest mismatch actual: %#v  expected: %#v", hDigest, gDigest)
		//t.Log(util.FormatBytes(title+"ProposedBlockHeader", header))
		t.Log(util.FormatBytes(title+"ProposedLEhash", []byte(hDigest[:])))

	}

	// check difficulty
	if hDigest.Cmp(bits.BigInt()) > 0 {
		t.Errorf("difficulty NOT met\n")
	}

	// compute block size
	blockSize := len(header) + len(base)

	// pack the block
	blk := blockrecord.PackedBlock(make([]byte, 0, blockSize))
	blk = append(blk, header...)
	blk = append(blk, base...)

	if !bytes.Equal(blk, gBlock) {
		t.Errorf("initial block assembly mismatch actual: %x  expected: %x", blk, gBlock)
		t.Log(util.FormatBytes(title+"GenesisBlock", blk))
	}

	t.Logf("packed block: %x", blk)

	for i := 0; i < len(blk); i += 16 {
		line := ""
		line += fmt.Sprintf("%08x ", i)
		text := ""
		for j := 0; j < 16; j += 1 {
			if 8 == j {
				line += " "
			}
			if i+j >= len(blk) {
				line += "   "
			} else {
				b := blk[i+j]
				line += fmt.Sprintf(" %02x", b)
				r := rune(b)
				if unicode.IsPrint(r) {
					text += string(r)
				} else {
					text += "."
				}
			}
		}

		t.Log(line + "  |" + text + "|")
	}

	// // unpack the block
	// unpacked, err := blk.Unpack()
	// if nil != err {
	// 	t.Fatalf("unpack block failed: error: %v", err)
	// }

	// if unpacked.Header.Timestamp != h.Timestamp {
	// 	t.Fatalf("block ntime mismatch: actual 0x%08x  expected 0x%08x", unpacked.Header.Timestamp, h.Timestamp)
	// }

	// // 	if unpacked.Timestamp != timestamp {
	// // 		t.Fatalf("block timestamp mismatch: actual %v  expected %v", unpacked.Timestamp, timestamp)
	// // 	}

	// t.Logf("unpacked block: %#v", unpacked)

	// ***** FIX THIS: is block.Pack() really necessary, need to determinal all the previous owners
	// ***** FIX THIS: suggest not; just keep tx in packed form
	//      // re-pack
	// 	reDigest, rePacked, ok := block.Pack(unpacked.Number, timestamp, &unpacked.Header.Bits, unpacked.Header.Time, unpacked.Header.Nonce, append(extraNonce1, extraNonce2...), unpacked.Addresses, unpacked.TxIds)

	// 	if !ok {
	// 		t.Fatal("block.Pack failed")
	// 	}

	// 	if reDigest != gDigest {
	// 		t.Fatalf("re-digest mismatch actual: %#v  expected: %#v", reDigest, gDigest)
	// 	}

	// 	if !bytes.Equal(rePacked, blk) {
	// 		t.Fatalf("re-packed mismatch actual: %x  expected: %x", rePacked, blk)
	// 	}

	// 	// log the final result
	// 	t.Logf("Genesis digest: %#v", reDigest)
	// 	t.Logf("Genesis block:  %x", rePacked)

	// 	// hex dumps for genesis.go
	// 	t.Log(util.FormatBytes(title+"GenesisBlock", rePacked))
	// 	t.Log(util.FormatBytes(title+"GenesisDigest", reDigest[:]))

	// 	// check that these match the current genesis block/digest
	// 	if reDigest != gDigest {
	// 		t.Fatalf("re-digest/Genesis mismatch actual: %#v  expected: %#v", reDigest, gDigest)
	// 	}

	// 	if !bytes.Equal(rePacked, gBlock) {
	// 		t.Fatalf("re-packed/Genesis mismatch actual: %x  expected: %x", rePacked, gBlock)
	// 	}
	// }

	// // test the real genesis block
	// func TestGenesisBlock(t *testing.T) {
	// 	doReal(t, "Live", block.LiveGenesisDigest, block.LiveGenesisBlock)
	// 	doReal(t, "Test", block.TestGenesisDigest, block.TestGenesisBlock)
	// }

	// func doReal(t *testing.T, title string, gDigest blockdigest.Digest, gBlock blockrecord.PackedHeader) {

	// 	// unpack the block
	// 	var unpacked block.Block
	// 	err := block.LiveGenesisBlock.Unpack(&unpacked)
	// 	if nil != err {
	// 		t.Fatalf("unpack block failed: error: %v", err)
	// 	}

	// 	if verboseTesting { // turn on in all_test.go
	// 		t.Logf("unpacked block: %v", unpacked)
	// 	}

	// 	// check current genesis digest matches
	// 	if unpacked.Digest != block.LiveGenesisDigest {
	// 		t.Fatalf("digest/Genesis mismatch actual: %#v  expected: %#v", unpacked.Digest, block.LiveGenesisDigest)
	// 	}

	// 	// check block number
	// 	if unpacked.Number != genesisBlockNumber {
	// 		t.Fatalf("block number: %d  expected %d", unpacked.Number, genesisBlockNumber)
	// 	}

	// 	// check the address matches
	// 	if 1 != len(unpacked.Addresses) {
	// 		t.Fatalf("Addresses: found: %d  expected: %d", len(unpacked.Addresses), 1)
	// 	}
	// 	if unpacked.Addresses[0].String() != genesisLiveRawAddress {
	// 		t.Fatalf("RawAddress: %q  expected: %q", unpacked.Addresses[0].String(), genesisLiveRawAddress)
	// 	}
}
