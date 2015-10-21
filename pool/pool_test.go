// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool_test

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/pool"
	"testing"
)

// helper to add to pool
func poolAdd(t *testing.T, p *pool.Pool, key string, data string) {
	p.Add([]byte(key), []byte(data))
}

// helper to remove from pool
func poolRemove(t *testing.T, p *pool.Pool, key string) {
	p.Remove([]byte(key))
}

// main pool test
func TestPool(t *testing.T) {
	setup(t)
	defer teardown(t)

	p := pool.New(pool.TestData)

	// ensure that pool was empty
	checkAgain(t, true)

	// add more items than poolSize
	poolAdd(t, p, "key-one", "data-one")
	poolAdd(t, p, "key-two", "data-two")
	poolAdd(t, p, "key-remove-me", "to be deleted")
	poolRemove(t, p, "key-remove-me")
	poolAdd(t, p, "key-three", "data-three")
	poolAdd(t, p, "key-one", "data-one")     // duplicate
	poolAdd(t, p, "key-three", "data-three") // duplicate
	poolAdd(t, p, "key-four", "data-four")
	poolAdd(t, p, "key-delete-this", "to be deleted")
	poolAdd(t, p, "key-five", "data-five")
	poolAdd(t, p, "key-six", "data-six")
	poolRemove(t, p, "key-delete-this")
	poolAdd(t, p, "key-seven", "data-seven")
	poolAdd(t, p, "key-one", "data-one(NEW)") // duplicate

	// ensure we get all of the pool
	data, err := p.Fetch([]byte{}, 20)
	if nil != err {
		t.Errorf("Error on Recent: %v", err)
		return
	}

	// this is the expected order
	check := makeElements([]stringElement{
		{"key-five", "data-five"},
		{"key-four", "data-four"},
		{"key-one", "data-one(NEW)"},
		{"key-seven", "data-seven"},
		{"key-six", "data-six"},
		{"key-three", "data-three"},
		{"key-two", "data-two"},
		// {"key-one", "data-one"}, // this was removed

	})

	// ensure lengths match
	if len(data) != len(check) {
		t.Errorf("Length mismatch, got: %d  expected: %d", len(data), len(check))
	}

	// compare all items from pool
	for i, a := range data {
		if i >= len(check) {
			t.Errorf("%d: Excess, got: '%s'  expected: Nothing", i, a)
		} else if !bytes.Equal(check[i].Key, a.Key) || !bytes.Equal(check[i].Value, a.Value) {
			t.Errorf("%d: Mismatch, got: '%s:%s'  expected: '%s:%s'", i,
				a.Key, a.Value,
				check[i].Key, check[i].Value)
		}
	}

	// retrieve a key not in the pool's cache
	d2, found := p.Get([]byte("key-two"))
	if !found {
		t.Errorf("not found")
	}
	e2 := "data-two"
	if string(d2) != e2 {
		t.Errorf("Mismatch on Get, got: '%s'  expected: '%s'", d2, e2)
	}

	// retrieve a key not in the pool
	dn, found := p.Get(nonExistantKey)
	if found {
		t.Errorf("Unexpected data on Get, got: '%s'  expected: nil", dn)
	}

	// recheck
	checkAgain(t, false)

	// check that restarting database keeps data
	pool.Finalise()
	pool.Initialise(databaseFileName)
	checkAgain(t, false)
}

func checkAgain(t *testing.T, empty bool) {

	// new pool, but same prefix so can access data entered above
	p := pool.New(pool.TestData)

	// cache will be empty
	data, err := p.Fetch([]byte{}, 5)
	if nil != err {
		t.Errorf("Error on Recent: %v", err)
		return
	}
	if empty && 0 != len(data) {
		t.Errorf("Pool cache was not empty, count = %d", len(data))
	}

	check := makeElements([]stringElement{
		{"key-one", "data-one(NEW)"},
		{"key-seven", "data-seven"},
		{"key-six", "data-six"},
		{"key-five", "data-five"},
		{"key-four", "data-four"},
		{"key-three", "data-three"},
		{"key-two", "data-two"},
		// {"key-one", "data-one"}, // this was removed
	})

	for i, e := range check {

		data, found := p.Get([]byte(e.Key))
		if empty {
			if found {
				t.Errorf("checkAgain: %d: Unexpected data on Get('%s'), got: '%s'  expected: nil", i, e.Key, data)
			}
		} else {
			if !found {
				t.Errorf("checkAgain: %d: Error on Get('%s') not found", i, e.Key)
			}
			if !bytes.Equal(data, e.Value) {
				t.Errorf("checkAgain: %d: Mismatch on Get('%s'), got: '%s'  expected: '%s'", i, e.Key, data, e.Value)
			}
		}
	}

	// attempt to retrieve a key that does not exist
	dn, found := p.Get(nonExistantKey)
	if found {
		t.Errorf("checkAgain: Unexpected data on Get('/nonexistant'), got: '%s'  expected: nil", dn)
	}
}
