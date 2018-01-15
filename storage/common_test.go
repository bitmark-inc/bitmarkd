// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage_test

import (
	"github.com/bitmark-inc/bitmarkd/storage"
	"os"
	"testing"
)

// test database file
const (
	databaseFileName = "test.leveldb"
)

// common test setup routines

// remove all files created by test
func removeFiles() {
	os.RemoveAll(databaseFileName)
}

// configure for testing
func setup(t *testing.T) {
	removeFiles()
	err := storage.Initialise(databaseFileName)
	if nil != err {
		t.Fatalf("storage initialise error: %s", err)
	}
}

// post test cleanup
func teardown(t *testing.T) {
	storage.Finalise()
	removeFiles()
}

// a string data item
type stringElement struct {
	key   string
	value string
}

// make an element array
func makeElements(input []stringElement) []storage.Element {
	output := make([]storage.Element, 0, len(input))
	for _, e := range input {
		output = append(output, storage.Element{
			Key:   []byte(e.key),
			Value: []byte(e.value),
		})
	}
	return output
}

// data for various test routines

// this is the expected order
var expectedElements = makeElements([]stringElement{
	{"key-five", "data-five"},
	{"key-four", "data-four"},
	{"key-one", "data-one(NEW)"},
	{"key-seven", "data-seven"},
	{"key-six", "data-six"},
	{"key-three", "data-three"},
	{"key-two", "data-two"},
	// {"key-one", "data-one"}, // this was removed
})

// a key that must not exist
var nonExistantKey = []byte("/nonexistant")

// sample key and data
var testKey = []byte("key-two")
var testData = "data-two"
