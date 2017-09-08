// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util_test

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"strings"
	"testing"
)

// Test IP address detection
func TestCanonical(t *testing.T) {

	type item struct {
		in  string
		out string
	}

	testData := []item{
		{"127.1:1234", "127.0.0.1:1234"},
		{"127.0.0.1:1234", "127.0.0.1:1234"},
		{"127.0.0.1:1", "127.0.0.1:1"},
		{" 127.0.0.1 : 1 ", "127.0.0.1:1"},
		{"127.0.0.1:65535", "127.0.0.1:65535"},
		{"0.0.0.0:1234", "0.0.0.0:1234"},
		{"[::1]:1234", "[::1]:1234"},
		{"[::]:1234", "[::]:1234"},
		{"[0:0::0:0]:1234", "[::]:1234"},
		{"[0:0:0:0::1]:1234", "[::1]:1234"},
		{"[2404:6800:4008:c07::66]:443", "[2404:6800:4008:c07::66]:443"},
		{"[2404:6800:4008:0c07:0000:0000:0000:0066]:443", "[2404:6800:4008:c07::66]:443"},
	}

	for i, d := range testData {

		// create a connection item
		c, err := util.NewConnection(d.in)
		if nil != err {
			t.Fatalf("NewConnection failed on:[%d] %q  error: %v", i, d.in, err)
		}

		// convert to text
		s, v6 := c.CanonicalIPandPort("")
		if s != d.out {
			t.Fatalf("failed on:[%d] %q  actual: %q  expected: %q", i, d.in, s, d.out)
		}

		t.Logf("converted:[%d]: %q  to(%t): %q", i, d.in, v6, s)

		// check packing/unpacking
		pk := c.Pack()
		cu, n := pk.Unpack()
		if len(pk) != n {
			t.Fatalf("Unpack failed on:[%d] %q  only read: %d of: %d bytes", i, d.in, n, len(pk))
		}
		su, v6u := cu.CanonicalIPandPort("")
		if su != s {
			t.Fatalf("failed on:[%d] %x  actual: %q  expected: %q", i, pk, su, s)
		}
		if v6u != v6 {
			t.Fatalf("failed on:[%d] %x  actual v6: %t  expected v6: %t", i, pk, v6u, v6)
		}
	}
}

// Test IP address
func TestCanonicalIP(t *testing.T) {

	testData := []string{
		"256.0.0.0:1234",
		"0.256.0.0:1234",
		"0.0.256.0:1234",
		"0.0.0.256:1234",
		"0:0:1234",
		"[]:1234",
		"[as34::]:1234",
		"[1ffff::]:1234",
		"[2404:6800:4008:0c07:0000:0000:0000:0066:1234]:443",
		"*:1234",
	}

	for i, d := range testData {
		c, err := util.NewConnection(d)
		if nil == err {
			s, v6 := c.CanonicalIPandPort("")
			t.Fatalf("eroneoulssly converted:[%d]: %q  to(%t): %q", i, d, v6, s)
		}
		if strings.Contains(err.Error(), "no such host") {
			// expected error
		} else if fault.ErrInvalidIPAddress != err {
			t.Fatalf("NewConnection failed on:[%d] %q  error: %v", i, d, err)
		}
	}
}

// Test port range
func TestCanonicalPort(t *testing.T) {

	testData := []string{
		"127.0.0.1:0",
		"127.0.0.1:65536",
		"127.0.0.1:-1",
	}

	for i, d := range testData {
		c, err := util.NewConnection(d)
		if nil == err {
			s, v6 := c.CanonicalIPandPort("")
			t.Fatalf("eroneoulssly converted:[%d]: %q  to(%t): %q", i, d, v6, s)
		}
		if fault.ErrInvalidPortNumber != err {
			t.Fatalf("NewConnection failed on:[%d] %q  error: %v", i, d, err)
		}
	}
}
