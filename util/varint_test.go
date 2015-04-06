// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util_test

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/util"
	"testing"
)

var varint64Tests = []struct {
	value   uint64
	encoded []byte
}{
	{0, []byte{0x00}},
	{1, []byte{0x01}},
	{127, []byte{0x7f}},
	{128, []byte{0x80, 0x01}},
	{137, []byte{0x89, 0x01}},
	{255, []byte{0xff, 0x01}},
	{256, []byte{0x80, 0x02}},
	{16383, []byte{0xff, 0x7f}},
	{16384, []byte{0x80, 0x80, 0x01}},
	{0x7fffffffffffffff, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}},
	{0x8000000000000000, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}},
	{0xfffffffffffffffe, []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
	{0xffffffffffffffff, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}},
}

var varint64TruncatedTests = [][]byte{
	{},
	{0x80},
	{0xff},
	{0x80, 0x80},
	{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
}

func TestToVarint64(t *testing.T) {

	for i, item := range varint64Tests {
		if result := util.ToVarint64(item.value); !bytes.Equal(result, item.encoded) {
			t.Errorf("%d: ToVarint64(%x) -> %x  expected: %x", i, item.value, result, item.encoded)
		}
	}
}

func TestFromVarint64(t *testing.T) {

	for i, item := range varint64Tests {
		result1, count1 := util.FromVarint64(item.encoded)
		if result1 != item.value {
			t.Errorf("%d: FromVarint64(%x) -> %d  expected: %d", i, item.encoded, result1, item.value)
		}

		b := item.encoded
		suffix := []byte{0xff, 0x97, 0x23}
		b = append(b, suffix...)

		result2, count2 := util.FromVarint64(item.encoded)
		if result2 != item.value || count1 != count2 {
			t.Errorf("%d: FromVarint64(%x) -> %d  expected: %d", i, b, result2, item.value)
		}
		if !bytes.Equal(suffix, b[count2:]) {
			t.Errorf("%d: suffix: %x  expected: %x", i, b[count2:], suffix)
		}
	}

	for i, item := range varint64TruncatedTests {
		result, count := util.FromVarint64(item)
		if 0 != result || 0 != count {
			t.Errorf("%d: FromVarint64(%x) -> %d, %d  expected: 0, 0", i, item, result, count)
		}
	}
}
