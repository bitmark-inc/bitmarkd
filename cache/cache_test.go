// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package cache

import (
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	Initialise()
	defer Finalise()

	Pool.TestB.Put("key-one", "data-one")
	Pool.TestB.Put("key-two", "data-two")
	Pool.TestB.Put("key-remove-me", "to be deleted")
	Pool.TestB.Delete("key-remove-me")
	Pool.TestB.Put("key-three", "data-three")
	Pool.TestB.Put("key-one", "data-one")     // duplicate
	Pool.TestB.Put("key-three", "data-three") // duplicate
	Pool.TestB.Put("key-four", "data-four")
	Pool.TestB.Put("key-delete-this", "to be deleted")
	Pool.TestB.Put("key-five", "data-five")
	Pool.TestB.Put("key-six", "data-six")
	Pool.TestB.Delete("key-delete-this")
	Pool.TestB.Put("key-seven", "data-seven")
	Pool.TestB.Put("key-one", "data-one(NEW)") // duplicate
	expectedItems := map[string]string{
		"key-one":   "data-one(NEW)",
		"key-two":   "data-two",
		"key-three": "data-three",
		"key-four":  "data-four",
		"key-five":  "data-five",
		"key-six":   "data-six",
		"key-seven": "data-seven",
	}

	if Pool.TestB.Size() != len(expectedItems) {
		t.Errorf("Length mismatch, got: %d  expected: %d", len(Pool.OrphanPayment.items), len(expectedItems))
	}

	for key, val := range Pool.TestB.Items() {
		expVal, ok := expectedItems[key]
		if !ok || val.(string) != expVal {
			t.Fail()
		}
	}
}

func TestExpiration(t *testing.T) {
	Initialise()
	defer Finalise()

	Pool.TestA.Put("a1", struct{}{})
	Pool.TestA.Put("a2", struct{}{})
	Pool.TestA.Put("a3", struct{}{})
	Pool.TestB.Put("b1", struct{}{})
	Pool.TestB.Put("b2", struct{}{})
	Pool.TestB.Put("b3", struct{}{})
	expectedKeysInPoolA := map[string]bool{"a1": false, "a2": false, "a3": false}
	expectedKeysInPoolB := map[string]bool{"b1": true, "b2": true, "b3": true}

	time.Sleep(3 * time.Second)
	deleteExpiredItems()

	for key, existed := range expectedKeysInPoolA {
		_, ok := Pool.TestA.Get(key)
		if ok != existed {
			t.Fatalf("the existence of key \"%s\" should be %t instead of %t", key, existed, ok)
		}
	}

	for key, existed := range expectedKeysInPoolB {
		_, ok := Pool.TestB.Get(key)
		if ok != existed {
			t.Fatalf("the existence of key \"%s\" should be %t instead of %t", key, existed, ok)
		}
	}
}
