// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package pool_test

import (
	"github.com/bitmark-inc/bitmarkd/pool"
	"os"
	"testing"
)

// test database file
const (
	databaseFileName = "test.leveldb"
)

// a key that must not exist
var nonExistantKey = []byte("/nonexistant")

// common test setup routines

// remove all files created by test
func removeFiles() {
	os.RemoveAll(databaseFileName)
}

// configure for testing
func setup(t *testing.T) {
	removeFiles()
	pool.Initialise(databaseFileName)
}

// post test cleanup
func teardown(t *testing.T) {
	pool.Finalise()
	removeFiles()
}

// a string data item
type stringElement struct {
	key   string
	value string
}

// make an element array
func makeElements(input []stringElement) []pool.Element {
	output := make([]pool.Element, 0, len(input))
	for _, e := range input {
		output = append(output, pool.Element{
			Key:   []byte(e.key),
			Value: []byte(e.value),
		})
	}
	return output
}
