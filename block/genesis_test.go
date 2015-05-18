// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"strings"
	"testing"
	"time"
)

// some constants embedded into the genesis block
const (
	genesisBlockNumber = uint64(1)
)

// some data embedded into the genesis block
var (
	// for live chain
	genesisLiveAddresses = []block.MinerAddress{
		{
			Currency: "",
			Address:  "Bitmark Genesis Block",
		},
	}
	genesisLiveRawAddress = "\x00\x15" + "Bitmark Genesis Block"

	// for testing chain
	genesisTestAddresses = []block.MinerAddress{
		{
			Currency: "",
			Address:  "Bitmark Testing Genesis Block",
		},
	}
	genesisTestRawAddress = "\x00\x1d" + "Bitmark Testing Genesis Block"
)

// create the live genesis block
//              worker       id, extranonce2,  ntime,      nonce.
// {"params": ["miner-two", "42", "01000000", "5478424c", "f0202692"], "id": 1, "method": "mining.submit"}
func TestLiveGenesisAssembly(t *testing.T) {

	// fixed data used to create genesis block
	// ---------------------------------------

	// nonce provided by statum sserver
	extraNonce1 := []byte("BMRK") // "424d524b"

	// nonces obtained from miner
	nonce := uint32(0x70295b74)
	extraNonce2 := []byte{0x01, 0x00, 0x00, 0x00}

	doCalc(t, "Live", extraNonce1, extraNonce2, nonce, genesisLiveAddresses, genesisLiveRawAddress, block.LiveGenesisDigest, block.LiveGenesisBlock)
}

// create the test genesis block
func TestTestGenesisAssembly(t *testing.T) {

	// fixed data used to create genesis block
	// ---------------------------------------

	// nonce provided by statum server
	extraNonce1 := []byte("BMRK") // "424d524b"

	// nonces obtained from miner
	nonce := uint32(0xaecca83b)
	extraNonce2 := []byte{0x01, 0x00, 0x00, 0x00}

	doCalc(t, "Test", extraNonce1, extraNonce2, nonce, genesisTestAddresses, genesisTestRawAddress, block.TestGenesisDigest, block.TestGenesisBlock)
}

func doCalc(t *testing.T, title string, extraNonce1 []byte, extraNonce2 []byte, nonce uint32, addresses []block.MinerAddress, rawAddress string, gDigest block.Digest, gBlock block.Packed) {

	// timestamp conversion

	// date -u -r $(printf '%d\n' 0x5478424b)
	// Fri Nov 28 09:37:15 UTC 2014
	// date -u -r $(printf '%d\n' 0x5478424b) '+%FT%TZ'
	// 2014-11-28T09:37:15Z

	timestamp, err := time.Parse(time.RFC3339, "2014-11-28T09:37:15Z")
	if nil != err {
		t.Fatalf("failed to parse time: err = %v", err)
	}
	timeUint64 := uint64(timestamp.UTC().Unix())
	ntime := uint32(0x5478424b)
	if timeUint64 != uint64(ntime) {
		t.Fatalf("time converted to: 0x%08x  expectd: %08x", timeUint64, ntime)
	}

	// some common static data
	version := uint32(2) // snapshot of version number
	previousBlock := block.Digest{}

	// Just calculations after this point
	// ----------------------------------

	coinbase := block.NewFullCoinbase(genesisBlockNumber, timestamp, append(extraNonce1, extraNonce2...), addresses)
	cDigest := block.NewDigest(coinbase)
	coinbaseLength := len(coinbase)

	transactionCount := 1

	// merkle tree
	tree := block.FullMerkleTree(cDigest, []block.Digest{})
	if tree[len(tree)-1] != cDigest {
		t.Fatalf("failed to compute tree: actual: %#v  expected: %#v", tree[len(tree)-1], cDigest)
	}

	// default difficulty
	bits := difficulty.New() // defaults to 1

	// block header
	h := block.Header{
		Version:       version,
		PreviousBlock: previousBlock,
		MerkleRoot:    tree[len(tree)-1],
		Time:          ntime,
		Bits:          *bits,
		Nonce:         nonce,
	}

	header := h.Pack()
	hDigest := header.Digest()

	// ok - log the header and coinbase data
	t.Logf("Title: %s", title)
	t.Logf("header: %#v\n", h)
	t.Logf("packed header: %x\n", header)
	t.Logf("coinbase: %x\n", coinbase)
	t.Logf("coinbase digest: %#v\n", cDigest)
	t.Logf("merkle tree: %#v\n", tree)
	t.Logf("hDigest: %#v\n", hDigest)
	t.Logf("hDigest little endian hex: %x\n", [block.DigestSize]byte(hDigest))

	// chack that it matches
	if hDigest != gDigest {
		t.Errorf("digest mismatch actual: %#v  expected: %#v", hDigest, gDigest)

		hexExtraNonce1 := fmt.Sprintf("%08x", extraNonce1)
		hexCoinbase := hex.EncodeToString(coinbase)
		n1 := strings.Index(hexCoinbase, hexExtraNonce1)
		n2 := n1 + 2*(len(extraNonce1)+len(extraNonce2)) // since 2 hex chars = 1 byte
		login := []struct {
			ID     interface{}   `json:"id"`
			Method string        `json:"method"`
			Result interface{}   `json:"result"`
			Error  []interface{} `json:"error"`
		}{
			{
				ID:     0,
				Method: "mining.subscribe",
				Result: []interface{}{
					[][]string{
						{"mining.set_difficulty", "1357"},
						{"mining.notify", "1234"},
					},
					hexExtraNonce1,
					len(extraNonce2),
				},
			},
			{
				ID:     "auth",
				Method: "mining.authorize",
				Result: true,
			},
		}

		requests := []struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
		}{
			{
				Method: "mining.set_difficulty",
				Params: []interface{}{
					difficulty.Current.Pdiff(),
				},
			},
			{
				Method: "mining.notify",
				Params: []interface{}{
					"42", // [0] job_id
					fmt.Sprintf("%s", previousBlock), // [1] previous link
					hexCoinbase[:n1],                 // [2] coinbase 1
					hexCoinbase[n2:],                 // [3] coinbase 2
					[]interface{}{},                  // [4] minimised merkle tree (empty)
					fmt.Sprintf("%08x", version),     // [5] version
					difficulty.Current.String(),      // [6] bits
					fmt.Sprintf("%08x", ntime),       // [7] time
					true, // [8] clean_jobs
				},
			},
		}

		for _, r := range login {
			b, err := json.Marshal(r)
			if nil != err {
				t.Errorf("json error: %v", err)
				return
			}

			t.Logf("JSON: %s", b)
		}
		for _, r := range requests {
			b, err := json.Marshal(r)
			if nil != err {
				t.Errorf("json error: %v", err)
				return
			}

			t.Logf("JSON: %s", b)
		}
	}

	// check difficulty
	if hDigest.Cmp(bits.BigInt()) > 0 {
		t.Fatalf("difficulty NOT met\n")
	}

	// compute block size
	blockSize := len(header) + 2 + coinbaseLength + 2 + len(tree)*block.DigestSize

	// pack the block
	blk := make([]byte, 0, blockSize)
	blk = append(blk, header...)
	blk = append(blk, byte(coinbaseLength&0xff))
	blk = append(blk, byte(coinbaseLength>>8))
	blk = append(blk, coinbase...)
	blk = append(blk, byte(transactionCount&0xff))
	blk = append(blk, byte(transactionCount>>8))

	buffer := new(bytes.Buffer)
	err = binary.Write(buffer, binary.LittleEndian, tree)
	if nil != err {
		t.Fatalf("binary.Write: err = %v", err)
	}

	blk = append(blk, buffer.Bytes()...)

	if len(blk) != blockSize {
		t.Fatalf("block size mismatch: actual: %d, expected: %d", len(blk), blockSize)
	}

	// unpack the block
	var unpacked block.Block
	err = block.Packed(blk).Unpack(&unpacked)
	if nil != err {
		t.Fatalf("unpack block failed: err = %v", err)
	}

	if unpacked.Header.Time != ntime {
		t.Fatalf("block ntime mismatch: actual 0x%08x  expected 0x%08x", unpacked.Header.Time, ntime)
	}

	if unpacked.Timestamp != timestamp {
		t.Fatalf("block timestamp mismatch: actual %v  expected %v", unpacked.Timestamp, timestamp)
	}

	t.Logf("unpacked block: %#v", unpacked)

	// re-pack
	reDigest, rePacked, ok := block.Pack(unpacked.Number, timestamp, &unpacked.Header.Bits, unpacked.Header.Time, unpacked.Header.Nonce, append(extraNonce1, extraNonce2...), unpacked.Addresses, unpacked.TxIds)

	if !ok {
		t.Fatal("block.Pack failed")
	}

	if reDigest != gDigest {
		t.Fatalf("re-digest mismatch actual: %#v  expected: %#v", reDigest, gDigest)
	}

	if !bytes.Equal(rePacked, blk) {
		t.Fatalf("re-packed mismatch actual: %x  expected: %x", rePacked, blk)
	}

	// log the final result
	if verboseTesting { // turn on in all_test.go
		t.Logf("Genesis digest: %#v", reDigest)
		t.Logf("Genesis block:  %x", rePacked)
	}

	// hex dumps for genesis.go
	t.Log(formatBytes(title+"GenesisBlock", rePacked))
	t.Log(formatBytes(title+"GenesisDigest", reDigest[:]))

	// check that these match the current genesis block/digest
	if reDigest != gDigest {
		t.Fatalf("re-digest/Genesis mismatch actual: %#v  expected: %#v", reDigest, gDigest)
	}

	if !bytes.Equal(rePacked, gBlock) {
		t.Fatalf("re-packed/Genesis mismatch actual: %x  expected: %x", rePacked, gBlock)
	}
}

// test the real genesis block
func TestGenesisBlock(t *testing.T) {
	doReal(t, "Live", block.LiveGenesisDigest, block.LiveGenesisBlock)
	doReal(t, "Test", block.TestGenesisDigest, block.TestGenesisBlock)
}

func doReal(t *testing.T, title string, gDigest block.Digest, gBlock block.Packed) {

	// unpack the block
	var unpacked block.Block
	err := block.LiveGenesisBlock.Unpack(&unpacked)
	if nil != err {
		t.Fatalf("unpack block failed: err = %v", err)
	}

	if verboseTesting { // turn on in all_test.go
		t.Logf("unpacked block: %v", unpacked)
	}

	// check current genesis digest matches
	if unpacked.Digest != block.LiveGenesisDigest {
		t.Fatalf("digest/Genesis mismatch actual: %#v  expected: %#v", unpacked.Digest, block.LiveGenesisDigest)
	}

	// check block number
	if unpacked.Number != genesisBlockNumber {
		t.Fatalf("block number: %d  expected %d", unpacked.Number, genesisBlockNumber)
	}

	// check the address matches
	if 1 != len(unpacked.Addresses) {
		t.Fatalf("Addresses: found: %d  expected: %d", len(unpacked.Addresses), 1)
	}
	if unpacked.Addresses[0].String() != genesisLiveRawAddress {
		t.Fatalf("RawAddress: %q  expected: %q", unpacked.Addresses[0].String(), genesisLiveRawAddress)
	}
}
