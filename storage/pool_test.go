// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage_test

import (
	"bytes"
	"crypto/rand"
	"github.com/bitmark-inc/bitmarkd/storage"
	"testing"
	"time"
)

// helper to add to pool
func poolPut(t *testing.T, p *storage.PoolHandle, key string, data string) {
	p.Put([]byte(key), []byte(data))
}

// helper to remove from pool
func poolDelete(t *testing.T, p *storage.PoolHandle, key string) {
	p.Delete([]byte(key))
}

// main pool test
func TestPool(t *testing.T) {
	setup(t)
	defer teardown(t)

	p := storage.Pool.TestData

	// ensure that pool was empty
	checkAgain(t, true)

	// add more items than poolSize
	poolPut(t, p, "key-one", "data-one")
	poolPut(t, p, "key-two", "data-two")
	poolPut(t, p, "key-remove-me", "to be deleted")
	poolDelete(t, p, "key-remove-me")
	poolPut(t, p, "key-three", "data-three")
	poolPut(t, p, "key-one", "data-one")     // duplicate
	poolPut(t, p, "key-three", "data-three") // duplicate
	poolPut(t, p, "key-four", "data-four")
	poolPut(t, p, "key-delete-this", "to be deleted")
	poolPut(t, p, "key-five", "data-five")
	poolPut(t, p, "key-six", "data-six")
	poolDelete(t, p, "key-delete-this")
	poolPut(t, p, "key-seven", "data-seven")
	poolPut(t, p, "key-one", "data-one(NEW)") // duplicate

	// ensure that data is correct
	checkResults(t, p)

	// recheck
	checkAgain(t, false)

	// check that restarting database keeps data
	storage.Finalise()
	storage.Initialise(databaseFileName)
	checkAgain(t, false)
}

func checkResults(t *testing.T, p *storage.PoolHandle) {

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

	p := storage.Pool.TestData

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

func TestWriteRead1(t *testing.T) {
	doWriteRead(t)
}
func TestWriteRead2(t *testing.T) {
	doWriteRead(t)
}
func TestWriteRead3(t *testing.T) {
	doWriteRead(t)
}
func TestWriteRead4(t *testing.T) {
	doWriteRead(t)
}

// main pool test
func doWriteRead(t *testing.T) {
	setup(t)
	defer teardown(t)

	p := storage.Pool.TestData

	key := rb(127)

	finish := time.After(90 * time.Second)
	stop := make(chan struct{})

	for j := 0; j < 10; j += 1 {
		go bg(&key, stop)
		go jr(&key, stop)
	}

	i := 0
loop:
	for {
		select {
		case <-finish:
			break loop
			//case <-time.After(1 * time.Millisecond):
		default:
		}

		i += 1

		oldkey := key
		key = rb(127)
		data := rb(156)

		p.Delete(key)
		d := p.Get(key)
		p.Put(key, data)

		p.Delete(oldkey)

		d = p.Get(key)
		if !bytes.Equal(data, d) {
			t.Errorf("%d: actual: %x  expected: %x", i, d, data)
		}

		d1 := p.Get(oldkey)
		if nil != d1 {
			t.Errorf("%d: actual: %x  expected: nil", i, d1)
		}
	}
	close(stop)
	time.Sleep(2 * time.Second)
}

func bg(key *[]byte, stop <-chan struct{}) {

	p := storage.Pool.TestData

	for {
		select {
		case <-stop:
			return
		//case <-time.After(t):
		default:
		}

		key2 := rb(129)
		data1 := rb(15)
		data2 := rb(165)

		p.Delete(key2)
		p.Put(key2, data1)
		p.Get(*key)
		p.Get(key2)
		p.Put(key2, data2)
		p.Get(key2)
		p.Get(*key)
	}
}

func jr(key *[]byte, stop <-chan struct{}) {

	p := storage.Pool.TestData

	for {
		select {
		case <-stop:
			return
		default:
			p.Get(*key)
		}
	}
}

func rb(n int) []byte {
	buffer := make([]byte, n)
	_, err := rand.Read(buffer)
	if nil != err {
		panic(err)
	}
	return buffer
}
