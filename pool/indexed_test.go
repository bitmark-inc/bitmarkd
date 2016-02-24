// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool_test

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/bitmark-inc/bitmarkd/pool"
	"testing"
)

// count the items to be expected in the pool
var indexedPoolAdds = 0
var indexedPoolRemoves = 0

// helper to add to pool
func indexedAdd(t *testing.T, p *pool.IndexedPool, key string, data string, newAddition bool) {
	indexedPoolAdds += 1
	justAdded, err := p.Add([]byte(key), []byte(data))
	if nil != err {
		t.Errorf("Error on add: %v", err)
	}
	if justAdded != newAddition {
		t.Errorf("Add: %s returned: %v  expected %v", key, justAdded, newAddition)
	}
}

// helper to remove from pool
func indexedRemove(t *testing.T, p *pool.IndexedPool, key string) {
	indexedPoolRemoves += 1
	err := p.Remove([]byte(key))
	if nil != err {
		t.Errorf("Error on remove: %v", err)
	}
}

// main pool test
func TestIndexedPool(t *testing.T) {
	setup(t)
	defer teardown(t)

	p := pool.NewIndexed('I')

	// ensure that pool was empty
	checkIndexedAgain(t, true)

	// add more items than poolSize
	indexedAdd(t, p, "key-one", "data-one", true)
	indexedAdd(t, p, "key-two", "data-two", true)
	indexedAdd(t, p, "key-remove-me", "to be deleted", true)
	indexedRemove(t, p, "key-remove-me")
	indexedAdd(t, p, "key-three", "data-three", true)
	indexedAdd(t, p, "key-one", "data-one", false)     // move to front
	indexedAdd(t, p, "key-three", "data-three", false) // move to front
	indexedAdd(t, p, "key-four", "data-four", true)
	indexedAdd(t, p, "key-delete-this", "to be deleted", true)
	indexedAdd(t, p, "key-five", "data-five", true)
	indexedAdd(t, p, "key-six", "data-six", true)
	indexedRemove(t, p, "key-delete-this")
	indexedAdd(t, p, "key-four", "data-four", false)
	indexedAdd(t, p, "key-seven", "data-seven", true)
	indexedAdd(t, p, "key-one", "data-one(NEW)", false) // move to front

	// ensure we get all of the pool
	// to used total of adds + removes + 1 (to ensure no extraeous data)
	start := gnomon.Cursor{}
	data, nextStart, err := p.Recent(&start, indexedPoolAdds+indexedPoolRemoves+1, converter)
	if nil != err {
		t.Errorf("Error on Recent: %v", err)
		return
	}

	if start == *nextStart {
		t.Errorf("no data, nextStart = %v", nextStart)
	}

	// basic length check ensure less than maximum
	maximumLength := indexedPoolAdds - indexedPoolRemoves
	if len(data) > maximumLength {
		t.Errorf("Length exceeds maximum, got: %d  expected maximum: %d", len(data), maximumLength)
	}

	// this is the expected order
	check := []stringElement{
		//{"key-one", "data-one"},
		{"key-two", "data-two"},
		{"key-three", "data-three"},
		{"key-five", "data-five"},
		{"key-six", "data-six"},
		{"key-four", "data-four"},
		{"key-seven", "data-seven"},
		{"key-one", "data-one(NEW)"},
	}

	// ensure lengths match exactly
	if len(data) != len(check) {
		t.Errorf("Length mismatch, got: %d  expected: %d", len(data), len(check))
	}

	// compare all items from pool
	for i, a1 := range data {
		a := a1.(stringElement)
		if i >= len(check) {
			t.Errorf("%d: Excess, got: '%s'  expected: Nothing", i, a)
		} else if check[i].key != a.key || check[i].value != a.value {
			t.Errorf("%d: Mismatch, got: '%s:%s'  expected: '%s:%s'", i,
				a.key, a.value,
				check[i].key, check[i].value)
		}
	}

	// retrieve a key not in the pool's cache
	d2, err := p.Get([]byte("key-two"))
	if nil != err {
		t.Errorf("Error on Get: %v", err)
	}
	e2 := "data-two"
	if string(d2) != e2 {
		t.Errorf("Mismatch on Get, got: '%s'  expected: '%s'", d2, e2)
	}

	// retrieve a key not in the pool
	dn, err := p.Get(nonExistantKey)
	if nil == err {
		t.Errorf("Unexpected data on Get, got: '%s'  expected: nil", dn)
	} else if !fault.IsErrNotFound(err) {
		t.Errorf("Error on Get: %v", err)
	}

	// recheck
	checkIndexedAgain(t, false)

	// check that restarting database keeps data
	pool.Finalise()
	pool.Initialise(databaseFileName)
	checkIndexedAgain(t, false)
}

func checkIndexedAgain(t *testing.T, empty bool) {

	// new pool, but same prefix so can access data entered above
	p := pool.NewIndexed('I')

	// is the mpool empty
	start := gnomon.Cursor{}
	data, nextStart, err := p.Recent(&start, 1, converter)
	if nil != err {
		t.Errorf("Error on Recent: %v", err)
		return
	}

	if empty {
		if start != *nextStart {
			t.Errorf("Still more data, nextStart = %v", nextStart)
		}
		if 0 != len(data) {
			t.Errorf("Pool cache was not empty, count = %d", len(data))
		}
	} else {
		if start == *nextStart {
			t.Errorf("No data in pool, nextStart = %v", nextStart)
		}
		if 0 == len(data) {
			t.Errorf("Pool cache was empty, count = %d", len(data))
		}
	}

	check := makeElements([]stringElement{
		//{"key-one", "data-one"},
		{"key-one", "data-one(NEW)"},
		{"key-two", "data-two"},
		{"key-three", "data-three"},
		{"key-four", "data-four"},
		{"key-five", "data-five"},
		{"key-six", "data-six"},
		{"key-seven", "data-seven"},
	})

	// check for existance
	for i, e := range check {

		data, err := p.Get(e.Key)
		if empty {
			if nil == err {
				t.Errorf("checkAgain: %d: Unexpected data on Get('%s'), got: '%s'  expected: nil", i, e.Key, data)
			} else if !fault.IsErrNotFound(err) {
				t.Errorf("Error on Get: %v", err)
			}
		} else {
			if nil != err {
				t.Errorf("checkAgain: %d: Error on Get('%s'): %v", i, e.Key, err)
			}
			if !bytes.Equal(data, e.Value) {
				t.Errorf("checkAgain: %d: Mismatch on Get('%s'), got: '%s'  expected: '%s'", i, e.Key, data, e.Value)
			}
		}
	}

	// attempt to retrieve a key that does not exist
	dn, err := p.Get(nonExistantKey)
	if nil == err {
		t.Errorf("checkAgain: Unexpected data on Get('/nonexistant'), got: '%s'  expected: nil", dn)
	} else if !fault.IsErrNotFound(err) {
		t.Errorf("Error on Get: %v", err)
	}
}

// convertion callback for REcent
func converter(key []byte, value []byte) interface{} {
	return stringElement{
		key:   string(key),
		value: string(value),
	}
}
