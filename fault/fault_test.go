// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fault

import (
	"testing"
)

var (
	ErrExistsOne   = existsError("exists one ")
	ErrExistsTwo   = existsError("exists two")
	ErrInvalidOne  = invalidError("invalid one")
	ErrInvalidTwo  = invalidError("invalid two")
	ErrLengthOne   = lengthError("length one")
	ErrLengthTwo   = lengthError("length two")
	ErrNotFoundOne = notFoundError("not found one")
	ErrNotFoundTwo = notFoundError("not found two")
	ErrProcessOne  = processError("process one")
	ErrProcessTwo  = processError("process two")
	ErrRecordOne   = recordError("record one")
	ErrRecordTwo   = recordError("record two")
)

// test that various not found errors can be subclassed
func TestAddress(t *testing.T) {
	errorList := []struct {
		err      error
		exists   bool
		invalid  bool
		length   bool
		notFound bool
		process  bool
		record   bool
	}{
		{ErrExistsOne, true, false, false, false, false, false},
		{ErrExistsTwo, true, false, false, false, false, false},
		{ErrInvalidOne, false, true, false, false, false, false},
		{ErrInvalidTwo, false, true, false, false, false, false},
		{ErrLengthOne, false, false, true, false, false, false},
		{ErrLengthTwo, false, false, true, false, false, false},
		{ErrNotFoundOne, false, false, false, true, false, false},
		{ErrNotFoundTwo, false, false, false, true, false, false},
		{ErrProcessOne, false, false, false, false, true, false},
		{ErrProcessTwo, false, false, false, false, true, false},
		{ErrRecordOne, false, false, false, false, false, true},
		{ErrRecordTwo, false, false, false, false, false, true},
	}

	for i, e := range errorList {
		err := e.err
		if IsErrExists(err) != e.exists {
			t.Errorf("%d: expected 'exists' == %v for err = %v", i, e.exists, err)
		}
		if IsErrInvalid(err) != e.invalid {
			t.Errorf("%d: expected 'invalid' == %v for err = %v", i, e.invalid, err)
		}
		if IsErrLength(err) != e.length {
			t.Errorf("%d: expected 'length' == %v for err = %v", i, e.length, err)
		}
		if IsErrNotFound(err) != e.notFound {
			t.Errorf("%d: expected 'not found' == %v for err = %v", i, e.notFound, err)
		}
		if IsErrProcess(err) != e.process {
			t.Errorf("%d: expected 'process' == %v for err = %v", i, e.process, err)
		}
		if IsErrRecord(err) != e.record {
			t.Errorf("%d: expected 'record' for err = %v", i, err)
		}
	}
}
