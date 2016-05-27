// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package util_test

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"testing"
)

// Test IP address detection
func TestCanonical(t *testing.T) {

	testData := []string{
		"127.0.0.1:1234",
		"127.0.0.1:1",
		" 127.0.0.1:1 ",
		"127.0.0.1:65535",
		"0.0.0.0:1234",
		"[::1]:1234",
		"[::]:1234",
		"[0:0::0:0]:1234",
		"[0:0:0:0::1]:1234",
		//"*:1234",
	}

	for i, d := range testData {
		c, err := util.CanonicalIPandPort("", d)
		if nil != err {
			t.Errorf("failed on:[%d] %q  err = %v", i, d, err)
			continue
		}
		t.Logf("converted:[%d]: %q  to: %q", i, d, c)
	}
}

// Test IP address
func TestCanonicalIP(t *testing.T) {

	testData := []string{
		"127.1:1234",
		"256.0.0.0:1234",
		"0.256.0.0:1234",
		"0.0.256.0:1234",
		"0.0.0.256:1234",
		"0:0:1234",
		"[]:1234",
		"[as34::]:1234",
		"[1ffff::]:1234",
		"*:1234",
	}

	for i, d := range testData {
		c, err := util.CanonicalIPandPort("", d)
		if fault.ErrInvalidIPAddress != err {
			t.Errorf("failed on:[%d] %q  err = %v", i, d, err)
			continue
		}
		t.Logf("converted:[%d]: %q  to: %q", i, d, c)
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
		c, err := util.CanonicalIPandPort("", d)
		if fault.ErrInvalidPortNumber != err {
			t.Errorf("failed on:[%d] %q  err = %v", i, d, err)
			continue
		}
		t.Logf("converted:[%d]: %q  to: %q", i, d, c)
	}
}
