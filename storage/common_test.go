// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"os"
	"testing"

	"github.com/bitmark-inc/logger"
)

// test database file
const (
	testingDirName   = "testing"
	databaseFileName = testingDirName + "/test"
)

// Test main entry-point
func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		os.Exit(1)
	}
	initialiseVars()
	result := m.Run()
	teardownTestDataAccess()
	teardown()
	os.Exit(result)
}

// remove all files created by test
func removeFiles() {
	_ = os.RemoveAll(testingDirName)
}

// configure for testing
func setup() error {
	removeFiles()
	_ = os.Mkdir(testingDirName, 0o700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      "testing.log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)

	// open database
	err := Initialise(databaseFileName, false)
	if err != nil {
		return fmt.Errorf("storage initialise error: %s", err.Error())
	}

	return nil
}

// post test cleanup
func teardown() {
	Finalise()
	logger.Finalise()
	removeFiles()
}

// a string data item
type stringElement struct {
	key   string
	value string
}

// make an element array
func makeElements(input []stringElement) []Element {
	output := make([]Element, 0, len(input))
	for _, e := range input {
		output = append(output, Element{
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
