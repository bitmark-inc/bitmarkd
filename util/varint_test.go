// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util_test

import (
	"bytes"
	"testing"

	"github.com/bitmark-inc/bitmarkd/util"
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

// bad values are treated as zero
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

	// expect result:0  and count:0
	for i, item := range varint64TruncatedTests {
		result, count := util.FromVarint64(item)
		if 0 != result || 0 != count {
			t.Errorf("%d: FromVarint64(%x) -> %d, %d  expected: 0, 0", i, item, result, count)
		}
	}
}

func TestClippedVarint64(t *testing.T) {

	var testItems = []struct {
		value   int
		count   int
		encoded []byte
		minimum int
		maximum int
	}{
		{0, 1, []byte{0x00}, 0, 1},
		{1, 1, []byte{0x01}, 0, 1},
		{127, 1, []byte{0x7f}, 1, 128},
		{128, 2, []byte{0x80, 0x01}, 1, 128},
		{0, 0, []byte{0x89, 0x01}, 1, 128},
		{137, 2, []byte{0x89, 0x01}, 1, 256},
		{255, 2, []byte{0xff, 0x01}, 1, 256},
		{256, 2, []byte{0x80, 0x02}, 1, 256},
		{0, 0, []byte{0x81, 0x02}, 1, 256},
		{257, 2, []byte{0x81, 0x02}, 1, 1024},
		{0, 0, []byte{0x81, 0x02}, 900, 1024},
		{0, 0, []byte{0xff, 0x7f}, 1024, 8192},
		{0, 0, []byte{0x80, 0x80, 0x01}, 1024, 8192},
		{16383, 2, []byte{0xff, 0x7f}, 1024, 65535},
		{16384, 3, []byte{0x80, 0x80, 0x01}, 1024, 65535},
		{65535, 3, []byte{0xff, 0xff, 0x03}, 1024, 65535},
		{0, 0, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f}, 1024, 8192},
		{0, 0, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}, 1024, 8192},
		{0, 0, []byte{0xfe, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 1024, 8192},
		{0, 0, []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 1024, 8192},
	}

	for i, item := range testItems {
		// incorrect range give error
		result0, count0 := util.ClippedVarint64(item.encoded, item.maximum, item.minimum)
		if 0 != result0 || 0 != count0 {
			t.Errorf("%d: ClipVarint64(%x) -> %d  expected: 0			", i, item.encoded, result0)
		}

		result1, count1 := util.ClippedVarint64(item.encoded, item.minimum, item.maximum)
		if count1 != item.count || result1 != item.value {
			t.Errorf("%d: ClipVarint64(%x) -> %d  count: %d  expected: %d  count: %d", i, item.encoded, result1, count1, item.value, item.count)
		}
	}
}
