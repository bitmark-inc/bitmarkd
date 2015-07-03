// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package difficulty_test

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"testing"
)

// test difficulty one
func TestFloatOne(t *testing.T) {

	expected := 1.0

	actual := difficulty.Current.Pdiff()

	if actual != expected {
		t.Errorf("actual: %f  expected: %f  diff: %g", actual, expected, actual-expected)
	}
}

type testItem struct {
	bits  uint32
	pdiff float64
	big   string
	bigf  string
}

var tests = []testItem{
	{
		pdiff: 1.0,
		bits:  0x1d00ffff,
		big:   "00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		bigf:  "00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	},
	{
		pdiff: 16307.669773817162,
		bits:  0x1b0404cb,
		big:   "00000000000404cb000000000000000000000000000000000000000000000000",
		bigf:  "00000000000404cb000000000831faf893b67b5d8fce7baed6a8f2ce5ab54f47",
	},
	{
		pdiff: 5.985742435503,
		bits:  0x1c2ac4af,
		big:   "000000002ac4af00000000000000000000000000000000000000000000000000",
		bigf:  "000000002ac4aefffffc829b8e92274bbff6c59550c329b2fbce485ecf691ff1",
	},
	{ // bitcoin block: 356030
		pdiff: 47644125009.457031, // bdiff = 47 643 398 017.803443
		bits:  0x181713dd,
		big:   "00000000000000001713dd000000000000000000000000000000000000000000",
		bigf:  "00000000000000001713dcffffffffe8d50f2300000017cf686c4944572dcd3f",
	},
}

// test 32 bit word
func TestUint32(t *testing.T) {

	d := difficulty.New()

	for i, item := range tests {

		d.SetBits(item.bits)
		actual := d.Pdiff()

		if actual != item.pdiff {
			t.Errorf("%d: actual: %f  pdiff: %f  diff: %g", i, actual, item.pdiff, actual-item.pdiff)
		}

		hexActual := d.String()
		hexExpected := fmt.Sprintf("%08x", item.bits)

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

	bits := []byte{0xcb, 0x04, 0x04, 0x1b} // little endian bytes
	expected := 16307.669773817162

	d.SetBytes(bits)
	actual := d.Pdiff()

	if actual != expected {
		t.Errorf("actual: %f  expected: %f  diff: %g", actual, expected, actual-expected)
	}
}

// test floating point (pdiff)
func TestPdiff(t *testing.T) {

	d := difficulty.New()

	for i, item := range tests {

		d.SetPdiff(item.pdiff)
		actual := d.Bits()

		if actual != item.bits {
			t.Errorf("%d: actual: 0x%08x  bits: 0x%08x", i, actual, item.bits)
		}

		hexActual := d.String()
		hexExpected := fmt.Sprintf("%08x", item.bits)

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
