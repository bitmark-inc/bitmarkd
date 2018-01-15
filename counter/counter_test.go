// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package counter_test

import (
	"github.com/bitmark-inc/bitmarkd/counter"
	"testing"
)

// test incrementing/decrementing a counter
func TestCounter(t *testing.T) {

	var c1 counter.Counter

	if !c1.IsZero() {
		t.Errorf("counter is not zero at start: %d", c1.Uint64())
	}

	c1.Increment()
	c1.Increment()
	c1.Increment()
	c1.Increment()
	c1.Increment()

	if 5 != c1.Uint64() {
		t.Errorf("counter is not 5 after iincrementing: %d", c1.Uint64())
	}

	c1.Decrement()

	if 4 != c1.Uint64() {
		t.Errorf("counter is not 5 after iincrementing: %d", c1.Uint64())
	}

	c1.Decrement()
	c1.Decrement()
	c1.Decrement()
	c1.Decrement()

	if !c1.IsZero() {
		t.Errorf("counter did not return to zero: %d", c1.Uint64())
	}

	c1.Decrement()

	// check against underflow, i.e. twos complement -1
	if ^uint64(0) != c1.Uint64() {
		t.Errorf("counter did not underflow: %d", c1.Uint64())
	}
}
