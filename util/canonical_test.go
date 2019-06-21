// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// Test IP address detection
func TestCanonical(t *testing.T) {

	type item struct {
		in  string
		out string
	}

	testData := []item{
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
		c, err := NewConnection(d.in)
		if nil != err {
			t.Fatalf("NewConnection failed on:[%d] %q  error: %s", i, d.in, err)
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
		c, err := NewConnection(d)
		if nil == err {
			s, v6 := c.CanonicalIPandPort("")
			t.Fatalf("eroneoulssly converted:[%d]: %q  to(%t): %q", i, d, v6, s)
		}
		if strings.Contains(err.Error(), "no such host") {
			// expected error
		} else if fault.ErrInvalidIpAddress != err {
			t.Fatalf("NewConnection failed on:[%d] %q  error: %s", i, d, err)
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
		c, err := NewConnection(d)
		if nil == err {
			s, v6 := c.CanonicalIPandPort("")
			t.Fatalf("eroneoulssly converted:[%d]: %q  to(%t): %q", i, d, v6, s)
		}
		if fault.ErrInvalidPortNumber != err {
			t.Fatalf("NewConnection failed on:[%d] %q  error: %s", i, d, err)
		}
	}
}

// helper
func makePacked(h string) PackedConnection {
	b, err := hex.DecodeString(h)
	if nil != err {
		panic(err)
	}
	return b
}

// Test of unpack
func TestCanonicalUnpack(t *testing.T) {

	type item struct {
		packed    PackedConnection
		addresses []string
		v4        string
		v6        string
	}

	testData := []item{
		{
			packed: makePacked("1304d200000000000000000000ffff7f0000011304d200000000000000000000000000000001"),
			addresses: []string{
				"127.0.0.1:1234",
				"[::1]:1234",
			},
			v4: "127.0.0.1:1234",
			v6: "[::1]:1234",
		},
		{
			packed: makePacked("1304d2000000000000000000000000000000011304d200000000000000000000ffff7f000001"),
			addresses: []string{
				"[::1]:1234",
				"127.0.0.1:1234",
			},
			v4: "127.0.0.1:1234",
			v6: "[::1]:1234",
		},
		{
			packed: makePacked("1301bb2404680040080c0700000000000000661301bb2404680040080c070000000000000066"),
			addresses: []string{
				"[2404:6800:4008:c07::66]:443",
				"[2404:6800:4008:c07::66]:443",
			},
			v4: "<nil>",
			v6: "[2404:6800:4008:c07::66]:443",
		},
		{ // extraneous data
			packed: makePacked("1301bb2404680040080c0700000000000000661301bb2404680040080c0700000000000000660000000000000000000000000000000000000000"),
			addresses: []string{
				"[2404:6800:4008:c07::66]:443",
				"[2404:6800:4008:c07::66]:443",
			},
			v4: "<nil>",
			v6: "[2404:6800:4008:c07::66]:443",
		},
		{ // bad data -> no items
			packed:    makePacked("1401bb2404680040080c0700000000000000661001bb2404680040080c0700000000000000660000000000000000000000000000000000000000"),
			addresses: []string{},
			v4:        "<nil>",
			v6:        "<nil>",
		},
		{ // bad data followed by good addresses -> consider as all bad
			packed:    makePacked("01221304d200000000000000000000ffff7f0000011304d200000000000000000000000000000001"),
			addresses: []string{},
			v4:        "<nil>",
			v6:        "<nil>",
		},
	}

	for i, data := range testData {
		p := data.packed
		a := data.addresses
		al := len(a)

		v4, v6 := p.Unpack46()
		v4s := "<nil>"
		if nil != v4 {
			v4s, _ = v4.CanonicalIPandPort("")
		}
		v6s := "<nil>"
		if nil != v6 {
			v6s, _ = v6.CanonicalIPandPort("")
		}
		if data.v4 != v4s {
			t.Errorf("unpack46:[%d]: v4 actual: %q  expected: %q", i, v4s, data.v4)
		}
		if data.v6 != v6s {
			t.Errorf("unpack66:[%d]: v6 actual: %q  expected: %q", i, v6s, data.v6)
		}

	inner:
		for k := 0; k < 10; k += 1 {
			l := len(p)
			c, n := p.Unpack()
			p = p[n:]

			if nil == c {
				// only signal error if nil was not just after last address
				if k != al {
					t.Errorf("unpack:[%d]: nil connection, n: %d", i, n)
				}

			} else {
				s, v6 := c.CanonicalIPandPort("")
				if k >= al {
					t.Errorf("unpack:[%d]: bytes: %d of %d result: (%t) %q", i, n, l, v6, s)
				} else if s != a[k] {
					t.Errorf("unpack:[%d]: bytes: %d of %d result: (%t) %q  expected: %s", i, n, l, v6, s, a[k])
				}
			}
			if n <= 0 {
				break inner
			}
		}
	}
}
