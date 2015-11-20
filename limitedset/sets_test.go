// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package limitedset_test

import (
	//"bytes"
	"github.com/bitmark-inc/bitmarkd/limitedset"
	"testing"
)

func TestAddition(t *testing.T) {

	items := []string{
		"0123456789",
		"abcdefghijklmnopqrstuvwxyz",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"hijklmn",
		"opqrstu",
		"vwxyzab",
		"cdefghi",
		"jklmnop",
		"qrstuvw",
	}

	expected := []string{
		"opqrstu",
		"vwxyzab",
		"cdefghi",
		"jklmnop",
		"qrstuvw",
	}

	check(t, items, expected)

}

// add a list of items and check that all the expected ones are present
// compute the ones that should not pe present and check that they are not
func check(t *testing.T, items []string, expected []string) {

	setSize := len(expected)

	s1 := limitedset.New(setSize)
	if nil == s1 {
		t.Fatalf("failed to create a limitedset of size: %d", setSize)
	}

	for _, d := range items {
		s1.Add(d)
	}

	hash := make(map[string]struct{}) // record all the expected

	// all expected must be present
	for i, d := range expected {
		hash[d] = struct{}{}
		if !s1.Exists(d) {
			t.Errorf("item[%d] missing: %q", i, d)
		}
	}

	// check the inputs (exclude the expected)
	for i, d := range items {
		if _, ok := hash[d]; ok {
			continue
		}
		if s1.Exists(d) {
			t.Errorf("item[%d] present: %q", i, d)
		}
	}
}

func TestPullToFront(t *testing.T) {

	items := []string{
		"0123456789",
		"abcdefghijklmnopqrstuvwxyz",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"abcdefg",
		"hijklmn",
		"abcdefg",
		"opqrstu",
		"abcdefg",
		"vwxyzab",
		"abcdefg",
		"cdefghi",
		"abcdefg",
		"jklmnop",
		"abcdefg",
		"abcdefg",
		"qrstuvw",
		"abcdefg",
		"xyzabcd",
	}

	expected := []string{
		"cdefghi",
		"jklmnop",
		"qrstuvw",
		"abcdefg",
		"xyzabcd",
	}

	check(t, items, expected)
}
