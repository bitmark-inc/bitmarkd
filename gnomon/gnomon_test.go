// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package gnomon_test

import (
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"testing"
)

// JSON test
func TestCursorJSON(t *testing.T) {

	cursor := gnomon.Cursor{}

	expectedB := "\"000000000000000000000000\""
	expectedItems := []string{
		expectedB,
		"\"0021436587bcadfe00004628\"",
		"\"0012345678abcdef00002468\"",
	}

	b, err := json.Marshal(cursor)
	if err != nil {
		t.Errorf("Error on json.Marshal: %v", err)
		return
	}

	if expectedB != string(b) {
		t.Errorf("json.Marshal returned: %s expected: %s", b, expectedB)
	}

	for i, expectedC := range expectedItems {

		in := []byte(expectedC)
		err = json.Unmarshal(in, &cursor)
		if err != nil {
			t.Errorf("Error on json.Unmarshal: %d: %v", i, err)
			return
		}

		actualC, err := json.Marshal(cursor)
		if err != nil {
			t.Errorf("Error on json.Marshal: %d: %v", i, err)
			return
		}

		if string(actualC) != expectedC {
			t.Errorf("json.Unmarshal: %d returned: %s expected: %s", i, actualC, expectedC)
		}
	}

	null := []byte("null")
	err = json.Unmarshal(null, &cursor)
	if err != nil {
		t.Errorf("Error on json.Unmarshal: %v", err)
		return
	}

	actualC, err := json.Marshal(cursor)
	if err != nil {
		t.Errorf("Error on json.Marshal: %v", err)
		return
	}

	if string(actualC) != expectedB {
		t.Errorf("json.Unmarshal returned: %s expected: %s", actualC, expectedB)
	}
}

// check adjacent calls do not return the same value
func TestCursorNotDuplicated(t *testing.T) {

	for i := 0; i < 10; i += 1 {
		cursor1 := gnomon.NewCursor()
		cursor2 := gnomon.NewCursor()
		cursor3 := gnomon.NewCursor()

		b1, err := json.Marshal(cursor1)
		if err != nil {
			t.Errorf("Error on json.Marshal: %v", err)
			return
		}
		b2, err := json.Marshal(cursor2)
		if err != nil {
			t.Errorf("Error on json.Marshal: %v", err)
			return
		}
		b3, err := json.Marshal(cursor3)
		if err != nil {
			t.Errorf("Error on json.Marshal: %v", err)
			return
		}

		s1 := string(b1)
		s2 := string(b2)
		s3 := string(b3)

		if s1 == s2 || s1 == s3 || s2 == s3 {
			t.Errorf("Error: duplicate cursors")
			t.Errorf("1: %s", s1)
			t.Errorf("2: %s", s2)
			t.Errorf("3: %s", s3)
		}
	}
}
