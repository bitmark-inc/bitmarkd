// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"bytes"
	"testing"
)

// helper to add to pool
func poolPut(p Handle, key string, data string) {
	p.Begin()
	p.Put([]byte(key), []byte(data), []byte{})
	p.Commit()
}

// helper to Remove from pool
func poolDelete(p Handle, key string) {
	p.Begin()
	p.Remove([]byte(key))
	p.Commit()
}

// main pool test
func TestPool(t *testing.T) {
	p := Pool.TestData

	// ensure that pool was empty
	checkAgain(t, true)

	// add more items than poolSize
	poolPut(p, "key-one", "data-one")
	poolPut(p, "key-two", "data-two")
	poolPut(p, "key-Remove-me", "to be deleted")
	poolDelete(p, "key-Remove-me")
	poolPut(p, "key-three", "data-three")
	poolPut(p, "key-one", "data-one")     // duplicate
	poolPut(p, "key-three", "data-three") // duplicate
	poolPut(p, "key-four", "data-four")
	poolPut(p, "key-delete-this", "to be deleted")
	poolPut(p, "key-five", "data-five")
	poolPut(p, "key-six", "data-six")
	poolDelete(p, "key-delete-this")
	poolPut(p, "key-seven", "data-seven")
	poolPut(p, "key-one", "data-one(NEW)") // duplicate

	// ensure that data is correct
	checkResults(t, p)

	// recheck
	checkAgain(t, false)

	// check that restarting database keeps data
	Finalise()
	_ = Initialise(databaseFileName, false)
	checkAgain(t, false)
}

func checkResults(t *testing.T, p Handle) {

	// ensure we get all of the pool
	cursor := p.NewFetchCursor()
	data, err := cursor.Fetch(20)
	if nil != err {
		t.Errorf("Error on Fetch: %v", err)
		return
	}

	// ensure lengths match
	if len(data) != len(expectedElements) {
		t.Errorf("Length mismatch, got: %d  expected: %d", len(data), len(expectedElements))
	}

	// compare all items from pool
	for i, a := range data {
		if i >= len(expectedElements) {
			t.Errorf("%d: Excess, got: '%s'  expected: Nothing", i, a)
		} else if !bytes.Equal(expectedElements[i].Key, a.Key) || !bytes.Equal(expectedElements[i].Value, a.Value) {
			t.Errorf("%d: Mismatch, got: '%s:%s'  expected: '%s:%s'", i,
				a.Key, a.Value,
				expectedElements[i].Key, expectedElements[i].Value)
		}
	}

	// retrieve 2 elements then next 2 - ensure no overlap
	cursor.Seek(nil)
	firstPair, err := cursor.Fetch(2)
	if nil != err {
		t.Errorf("Error on Fetch: %v", err)
		return
	}
	secondPair, err := cursor.Fetch(2)
	if nil != err {
		t.Errorf("Error on Fetch: %v", err)
		return
	}
	if bytes.Equal(firstPair[1].Key, secondPair[0].Key) {
		t.Errorf("Fetch Overlap got duplicate: '%s:%s'", firstPair[1].Key, firstPair[1].Value)
	}

	// check key exists
	if !p.Has(testKey) {
		t.Errorf("not found: %q", testKey)
	}

	// retrieve a key
	d2 := p.Get(testKey)
	if nil == d2 {
		t.Errorf("not found: %q", testKey)
	}
	if string(d2) != testData {
		t.Errorf("Mismatch on Get, got: '%s'  expected: '%s'", d2, testData)
	}

	// check that key does not exist
	if p.Has([]byte(nonExistantKey)) {
		t.Errorf("unexpectedly found: %q", nonExistantKey)
	}

	// retrieve a key not in the pool
	dn := p.Get(nonExistantKey)
	if nil != dn {
		t.Errorf("Unexpected data on Get, got: '%s'  expected: nil", dn)
	}
}

func checkAgain(t *testing.T, empty bool) {

	p := Pool.TestData

	// cache will be empty
	cursor := p.NewFetchCursor()
	data, err := cursor.Fetch(100) // all data
	if nil != err {
		t.Errorf("Error on Fetch: %v", err)
		return
	}
	if empty && 0 != len(data) {
		t.Errorf("Pool was not empty, count = %d", len(data))
	}

	for i, e := range expectedElements {

		data := p.Get([]byte(e.Key))
		if empty {
			if nil != data {
				t.Errorf("checkAgain: %d: Unexpected data on Get('%s'), got: '%s'  expected: nil", i, e.Key, data)
			}
		} else {
			if nil == data {
				t.Errorf("checkAgain: %d: Error on Get('%s') not found", i, e.Key)
			}
			if !bytes.Equal(data, e.Value) {
				t.Errorf("checkAgain: %d: Mismatch on Get('%s'), got: '%s'  expected: '%s'", i, e.Key, data, e.Value)
			}
		}
	}

	// try to retrieve some more data - shout be zero
	data, err = cursor.Fetch(100)
	if nil != err {
		t.Errorf("Error on Fetch: %v", err)
		return
	}
	n := len(data)
	if 0 != n {
		t.Errorf("checkAgain: extra: %d elements found", n)
		t.Errorf("checkAgain: data: %s", data)
	}

	// check that key does not exist
	if p.Has([]byte(nonExistantKey)) {
		t.Errorf("unexpectedly found: %q", nonExistantKey)
	}

	// attempt to retrieve a key that does not exist
	dn := p.Get(nonExistantKey)
	if nil != dn {
		t.Errorf("checkAgain: Unexpected data on Get('/nonexistant'), got: '%s'  expected: nil", dn)
	}
}

// since use batch write, should avoid cases for multiple write
