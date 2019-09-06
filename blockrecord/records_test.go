// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockrecord_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/blockrecord/mocks"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/stretchr/testify/assert"
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
	if nil != err {
		t.Fatalf("hex decode string error: %s", err)
	}

	d := blockdigest.NewDigest(leBinaryBlock)

	var expected blockdigest.Digest
	n, err := fmt.Sscan(r.beExpectedDigest, &expected)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
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
		t.Fatalf("hex(%s) to block digest error: %s", s, err)
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
		t.Fatalf("hex(%s) to merkle digest error: %s", s, err)
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
			t.Fatalf("fromLE hex error: %s", err)
		}
		b := bytes.NewBuffer(h)
		n := len(h)
		switch n {
		case 2:
			n := uint16(0)
			err = binary.Read(b, binary.LittleEndian, &n)
			if nil != err {
				t.Fatalf("fromLE read error: %s", err)
			}
			return uint64(n)
		case 4:
			n := uint32(0)
			err = binary.Read(b, binary.LittleEndian, &n)
			if nil != err {
				t.Fatalf("fromLE read error: %s", err)
			}
			return uint64(n)
		case 8:
			n := uint64(0)
			err = binary.Read(b, binary.LittleEndian, &n)
			if nil != err {
				t.Fatalf("fromLE read error: %s", err)
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

	d := blockdigest.NewDigest(p[:])

	var expected blockdigest.Digest
	n, err := fmt.Sscan(r.beExpectedDigest, &expected)
	if nil != err {
		t.Fatalf("hex to link error: %s", err)
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

	// marshal to JSON
	j, err := json.Marshal(h)
	if nil != err {
		t.Fatalf("marshal to JSON error: %s", err)
	}

	je := `{"version":1,"transactionCount":1,"number":"32","previousBlock":"81cd02ab7e569e8bcd9317e2fe99f2de44d49ab2b8851ba4a308000000000000","merkleRoot":"e320b6c2fffc8d750423db8b1eb942ae710e951ed797f7affc8892b0f1fc122b","timestamp":"1305998791","difficulty":"f2b9441a3243250d","nonce":"42a1469535a7d421"}`

	if je != string(j) {
		t.Fatalf("JSON mismatch: actual: %s  expected: %s", j, je)
	}

	// unmarshal json
	var uHeader blockrecord.Header
	err = json.Unmarshal(j, &uHeader)
	if nil != err {
		t.Fatalf("unmarshal from JSON error: %s", err)
	}

	if !reflect.DeepEqual(uHeader, h) {
		t.Fatalf("JSON mismatch: actual: %v  expected: %v", uHeader, h)
	}

	// check that truncated records give error
	// note: this stops at 1 less than block header size
	// so will never give a non-error response
loop:
	for i := 0; i < len(p); i += 1 {
		// test the unpacker with bad records
		h, _, _, err := blockrecord.ExtractHeader(p[:i], h.Number)
		if nil != err {
			continue loop
		}
		t.Errorf("unpack: unexpected success: header[:%d]: %+v", i, h)
	}
}

func TestResetDifficulty(t *testing.T) {
	difficulty.Current.Set(2)
	blockrecord.ResetDifficulty()

	difficulty := difficulty.Current.Value()
	assert.Equal(t, float64(1), difficulty, "not reset difficulty")
}

func TestDigestFromHashPool(t *testing.T) {
	ctl := gomock.NewController(t)
	mock := mocks.NewMockHandle(ctl)

	blockNumber := []byte{100}
	digestBytes := []byte{
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
	}

	var expectedDigest blockdigest.Digest
	_ = blockdigest.DigestFromBytes(&expectedDigest, digestBytes)

	mock.EXPECT().Empty().Return(false).Times(1)
	mock.EXPECT().Get(blockNumber).Return(digestBytes).Times(1)

	digest := blockrecord.DigestFromHashPool(mock, blockNumber)
	assert.Equal(t, expectedDigest, digest, "wrong digest")
}

func TestDigestFromCache(t *testing.T) {
	ctl := gomock.NewController(t)
	mock := mocks.NewMockHandle(ctl)

	blockNumber := []byte{100}
	digestBytes := []byte{
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
		0x11, 0x11, 0x11, 0x11,
	}

	var expectedDigest blockdigest.Digest
	_ = blockdigest.DigestFromBytes(&expectedDigest, digestBytes)

	mock.EXPECT().Empty().Return(false).Times(1)
	mock.EXPECT().Get(blockNumber).Return(digestBytes).Times(1)

	digest := blockrecord.DigestFromHashPool(mock, blockNumber)
	assert.Equal(t, expectedDigest, digest, "wrong digest")
}
