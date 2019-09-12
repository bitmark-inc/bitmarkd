// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package difficulty_test

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"

	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/stretchr/testify/assert"
)

// test difficulty initiialisation
func TestInitialBits(t *testing.T) {

	expected := difficulty.OneUint64

	actual := difficulty.Current.Bits()

	if actual != expected {
		t.Errorf("actual: %d  expected: %d", actual, expected)
	}
}

type testItem struct {
	bits       uint64
	reciprocal float64
	big        string
	bigf       string
}

var tests = []testItem{
	{
		reciprocal: 1,
		bits:       0x00ffffffffffffff,
		big:        "00ffffffffffffff800000000000000000000000000000000000000000000000",
		bigf:       "00ffffffffffffff800000000000000000000000000000000000000000000000",
	},
	{
		reciprocal: 2,
		bits:       0x01ffffffffffffff,
		big:        "007fffffffffffffc00000000000000000000000000000000000000000000000",
		bigf:       "007fffffffffffffc00000000000000000000000000000000000000000000000",
	},
	{
		reciprocal: 16,
		bits:       0x04ffffffffffffff,
		big:        "000ffffffffffffff80000000000000000000000000000000000000000000000",
		bigf:       "000ffffffffffffff80000000000000000000000000000000000000000000000",
	},
	{
		reciprocal: 256,
		bits:       0x08ffffffffffffff,
		big:        "0000ffffffffffffff8000000000000000000000000000000000000000000000",
		bigf:       "0000ffffffffffffff8000000000000000000000000000000000000000000000",
	},
	{
		reciprocal: 511.99999999999997,
		bits:       0x0800000000000007,
		big:        "0000800000000000038000000000000000000000000000000000000000000000",
		bigf:       "000080000000000003c000000000001e000000000000f0000000000007800000",
	},
	{
		reciprocal: 1000,
		bits:       0x090624dd2f1a9fbd,
		big:        "00004189374bc6a7ef4000000000000000000000000000000000000000000000",
		bigf:       "00004189374bc6a7ef7ced916872b020c49ba5e353f7ced916872b020c49ba5e",
	},
	{
		reciprocal: 10000,
		bits:       0x0da36e2eb1c432c9,
		big:        "0000068db8bac710cb2400000000000000000000000000000000000000000000",
		bigf:       "0000068db8bac710cb2617c1bda5119ce075f6fd21ff2e48e8a71de69ad42c3c",
	},
	{
		reciprocal: 47643398017.803443,
		bits:       0x23713f413f413f40,
		big:        "00000000001713f413f413f40000000000000000000000000000000000000000",
		bigf:       "00000000001713f413f413f40821936d0ab882041e769aaa6e7999e3a827ef1b",
	},
	{
		reciprocal: 1E15,
		bits:       0x31203af9ee756159,
		big:        "00000000000000480ebe7b9d5856400000000000000000000000000000000000",
		bigf:       "00000000000000480ebe7b9d585648806f5db1f9cfcec44485b1756799f713b1",
	},
	{
		reciprocal: 9223372036854775808,
		bits:       0x3fffffffffffffff,
		big:        "000000000000000001ffffffffffffff00000000000000000000000000000000",
		bigf:       "000000000000000001ffffffffffffff00000000000000000000000000000000",
	},
	{
		reciprocal: 18446744073709551616,
		bits:       0x40ffffffffffffff,
		big:        "000000000000000000ffffffffffffff80000000000000000000000000000000",
		bigf:       "000000000000000000ffffffffffffff80000000000000000000000000000000",
	},
	{
		reciprocal: 36893488147419099136,
		bits:       0x4000000000000007,
		big:        "0000000000000000008000000000000380000000000000000000000000000000",
		bigf:       "00000000000000000080000000000003c000000000001e000000000000f00000",
	},
	{
		reciprocal: 3138550867693340381917894711603833208051177722232017256448,
		bits:       0xbfffffffffffffff,
		big:        "00000000000000000000000000000000000000000000000001ffffffffffffff",
		bigf:       "00000000000000000000000000000000000000000000000001ffffffffffffff",
	},
	{ // the smallest value allowed (panics if smaller) = hash with 24 leading zero bytes!
		reciprocal: 6277101735386680066937501969125693243111159424202737451008,
		bits:       0xbf00000000000007,
		big:        "0000000000000000000000000000000000000000000000000100000000000007",
		bigf:       "0000000000000000000000000000000000000000000000000100000000000007",
	},
	// { // 13 - the theoretical smallest possible non-zero value - not useful
	// 	reciprocal: ?,
	// 	bits:       0xf7ffffffffffffff,
	// 	big:        "0000000000000000000000000000000000000000000000000000000000000001",
	// 	bigf:       "0000000000000000000000000000000000000000000000000000000000000001",
	// },
}

// test 64 bit word
func TestUint64(t *testing.T) {

	d := difficulty.New()

	for i, item := range tests {

		d.SetBits(item.bits)
		actual := d.Value()

		if actual != item.reciprocal {
			t.Errorf("%d: actual: %20.10f  reciprocal: %20.10f  diff: %g", i, actual, item.reciprocal, actual-item.reciprocal)
		}

		hexActual := d.String()
		hexExpected := fmt.Sprintf("%016x", item.bits)

		if hexActual != hexExpected {
			t.Errorf("%d: hex: actual: %q  expected: %q", i, hexActual, hexExpected)
		}

		bigActual := fmt.Sprintf("%064x", d.BigInt())

		if bigActual != item.big {
			t.Errorf("%d: big: actual: %q  expected: %q", i, bigActual, item.big)
		}
	}

}

// test bytes
func TestBytes(t *testing.T) {

	d := difficulty.New()

	// 0x0da36e2eb1c432c9
	bits := []byte{0xc9, 0x32, 0xc4, 0xb1, 0x2e, 0x6e, 0xa3, 0x0d} // little endian bytes
	d.SetBytes(bits)

	expected := float64(10000)
	actual := d.Value()

	bits2 := d.Bits()

	if math.Abs(actual-expected) > 0.000001 {
		t.Errorf("0x%016x:  actual: %f  expected: %f  diff: %g", bits2, actual, expected, actual-expected)
	}
}

// test JSON
func TestJSON(t *testing.T) {

	d := difficulty.New()

	for i, item := range tests {

		d.SetBits(item.bits)

		buffer, err := json.Marshal(d)
		if nil != err {
			t.Fatalf("%d: JSON encode error: %s", i, err)
		}

		dNew := difficulty.New()
		err = json.Unmarshal(buffer, dNew)
		if nil != err {
			t.Fatalf("%d: JSON decode error: %s", i, err)
		}

		actual := dNew.Bits()
		expected := item.bits

		if actual != expected {
			t.Errorf("%d: JSON actual: %016x  expected: %016x", i, actual, expected)
		}
	}
}

// test floating point (reciprocal)
func TestReciprocal(t *testing.T) {

	d := difficulty.New()

	for i, item := range tests {

		d.Set(item.reciprocal)
		actual := d.Bits()

		if actual != item.bits {
			t.Errorf("%d: actual: 0x%016x  bits: 0x%016x", i, actual, item.bits)
		}

		hexActual := d.String()
		hexExpected := fmt.Sprintf("%016x", item.bits)

		if hexActual != hexExpected {
			t.Errorf("%d: hex: actual: %q  expected: %q", i, hexActual, hexExpected)
		}

		bigActual := fmt.Sprintf("%064x", d.BigInt())

		if bigActual != item.bigf {
			t.Errorf("%d: big: actual: %q  expected: %q", i, bigActual, item.bigf)
		}
		bitsString := fmt.Sprintf("%v", d)
		if bitsString != hexExpected {
			t.Errorf("%d: String(): actual: %q  expected: %q", i, bitsString, hexExpected)
		}

		bigString := fmt.Sprintf("%#v", d)
		if bigString != item.bigf {
			t.Errorf("%d: GoString(): actual: %v  expected: %q", i, bigString, item.bigf)
		}
	}
}

func TestPrevTimespanBlockBeginAndEndWhenAtMiddle(t *testing.T) {
	height := uint64(difficulty.AdjustTimespanInBlocks*3 + 10)
	begin, end := difficulty.PrevTimespanBlockBeginAndEnd(height)

	assert.Equal(t, uint64(difficulty.AdjustTimespanInBlocks*3-1-difficulty.AdjustTimespanInBlocks), begin, "fail to get begin block")
	assert.Equal(t, uint64(difficulty.AdjustTimespanInBlocks*3-1), end, "get end block at middle")
}

func TestPrevTimespanBlockBeginAndEndWhenAtStart(t *testing.T) {
	height := uint64(difficulty.AdjustTimespanInBlocks * 3)
	begin, end := difficulty.PrevTimespanBlockBeginAndEnd(height)

	assert.Equal(t, uint64(difficulty.AdjustTimespanInBlocks*3-1-difficulty.AdjustTimespanInBlocks), begin, "fail to get begin block")
	assert.Equal(t, uint64(difficulty.AdjustTimespanInBlocks*3-1), end, "get end block at start")
}

func TestPrevTimespanBlockBeginAndEndWhenInFirstTimespan(t *testing.T) {
	height := uint64(difficulty.AdjustTimespanInBlocks + 10)
	begin, end := difficulty.PrevTimespanBlockBeginAndEnd(height)

	assert.Equal(t, uint64(2), begin, "fail to get begin block")
	assert.Equal(t, uint64(difficulty.AdjustTimespanInBlocks-1), end, "get end block in first timespan")
}

func TestHashrate(t *testing.T) {
	difficulty.Current.Set(4)
	hashrate := difficulty.Hashrate()

	// difficulty 4, log2(4) = 2
	// total bits of empty zero will 8+2 = 10 bits
	// possible hashes for a correct one is pow(2, 10) = 1024 hashes
	// expected time for a block is 120 seconds
	// hash rate = hashes / time = 1024 / 120
	expected := math.Floor((float64(1024)/120)*1000) / 1000
	assert.Equal(t, expected, hashrate, "network hashrate")
}

func TestNextDifficultyByPreviousTimespanWhenTooLong(t *testing.T) {
	diff := float64(8)
	targetTimespan := 2 * 60 * 200
	testTime := targetTimespan * 8
	actual := difficulty.NextDifficultyByPreviousTimespan(uint64(testTime), diff)

	assert.Equal(t, diff/4, actual, "wrong difficulty adjust")
}

func TestNextDifficultyByPreviousTimespanWhenTooShort(t *testing.T) {
	diff := float64(8)
	targetTimespan := 2 * 60 * 200
	testTime := targetTimespan / 8
	actual := difficulty.NextDifficultyByPreviousTimespan(uint64(testTime), diff)

	assert.Equal(t, diff*4, actual, "wrong difficulty adjust")
}

func TestNextDifficultyByPreviousTimespanWhenLarger(t *testing.T) {
	diff := float64(8)
	targetTimespan := 2 * 60 * 200
	testTime := targetTimespan * 3
	actual := difficulty.NextDifficultyByPreviousTimespan(uint64(testTime), diff)

	assert.Equal(t, diff/3, actual, "wrong difficulty adjust")
}

func TestNextDifficultyByPreviousTimespanWhenSmaller(t *testing.T) {
	diff := float64(8)
	targetTimespan := 2 * 60 * 200
	testTime := targetTimespan / 3
	actual := difficulty.NextDifficultyByPreviousTimespan(uint64(testTime), diff)

	assert.Equal(t, diff*3, actual, "wrong difficulty adjust")
}
