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

	actual := difficulty.Current.Float64()

	if actual != expected {
		t.Errorf("actual: %f  expected: %f  diff: %g", actual, expected, actual-expected)
	}
}

// test 32 bit word
func TestUint32(t *testing.T) {

	d := difficulty.New()

	value := uint32(0x1b0404cb)
	expected := 16307.669773817162

	d.SetUint32(value)
	actual := d.Float64()

	if actual != expected {
		t.Errorf("actual: %f  expected: %f  diff: %g", actual, expected, actual-expected)
	}

	hexActual := d.String()
	hexExpected := fmt.Sprintf("%08x", value)

	if hexActual != hexExpected {
		t.Errorf("hex: actual: %q  expected: %q", hexActual, hexExpected)
	}

	// a second test

	value = uint32(0x1c2ac4af)
	expected = 5.985742435503

	d.SetUint32(value)
	actual = d.Float64()

	if actual != expected {
		t.Errorf("actual: %f  expected: %f  diff: %g", actual, expected, actual-expected)
	}

	hexActual = d.String()
	hexExpected = fmt.Sprintf("%08x", value)

	if hexActual != hexExpected {
		t.Errorf("hex: actual: %q  expected: %q", hexActual, hexExpected)
	}

}

// test bytes
func TestBytes(t *testing.T) {

	d := difficulty.New()

	value := []byte{0xcb, 0x04, 0x04, 0x1b} // little endian bytes
	expected := 16307.669773817162

	d.SetBytes(value)
	actual := d.Float64()

	if actual != expected {
		t.Errorf("actual: %f  expected: %f  diff: %g", actual, expected, actual-expected)
	}
}
