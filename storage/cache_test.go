// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"testing"
)

func setupTestCache() Cache {
	return newCache()
}

func isSameByteSlice(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}

func TestWriteThenRead(t *testing.T) {
	cache := setupTestCache()

	key := "test"
	expected := []byte{'a', 'b', 'c', 'd'}

	actual, found := cache.Get(key)

	if found {
		t.Errorf("error key %s already exist value %v\n", key, actual)
	}

	cache.Set(dbPut, key, expected)
	actual, found = cache.Get(key)

	if !found || !isSameByteSlice(actual, expected) {
		t.Errorf("error set key %s, expect %v but get %v\n", key, expected, actual)
	}
}

func TestClear(t *testing.T) {
	cache := setupTestCache()

	key := "test"
	data := []byte{'a', 'b', 'c', 'd'}

	cache.Set(dbPut, key, data)
	cache.Clear()

	_, found := cache.Get(key)
	if found {
		t.Errorf("error Clear not working, expect cache is empty but not")
	}
}

func TestReadDeleteOperation(t *testing.T) {
	cache := setupTestCache()

	key := "test"
	data := []byte{'a', 'b', 'c', 'd'}

	cache.Set(dbDelete, key, data)

	_, found := cache.Get(key)
	if found {
		t.Errorf("delete operation should get nothing")
	}
}
