// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package fault_test

import (
	"testing"

	"github.com/bitmark-inc/bitmarkd/fault"
)

var (
	ErrExistsOne   = fault.ExistsError("exists one ")
	ErrExistsTwo   = fault.ExistsError("exists two")
	ErrInvalidOne  = fault.InvalidError("invalid one")
	ErrInvalidTwo  = fault.InvalidError("invalid two")
	ErrLengthOne   = fault.LengthError("length one")
	ErrLengthTwo   = fault.LengthError("length two")
	ErrNotFoundOne = fault.NotFoundError("not found one")
	ErrNotFoundTwo = fault.NotFoundError("not found two")
	ErrProcessOne  = fault.ProcessError("process one")
	ErrProcessTwo  = fault.ProcessError("process two")
	ErrRecordOne   = fault.RecordError("record one")
	ErrRecordTwo   = fault.RecordError("record two")
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
		if fault.IsErrExists(err) != e.exists {
			t.Errorf("%d: expected 'exists' == %v for err = %v", i, e.exists, err)
		}
		if fault.IsErrInvalid(err) != e.invalid {
			t.Errorf("%d: expected 'invalid' == %v for err = %v", i, e.invalid, err)
		}
		if fault.IsErrLength(err) != e.length {
			t.Errorf("%d: expected 'length' == %v for err = %v", i, e.length, err)
		}
		if fault.IsErrNotFound(err) != e.notFound {
			t.Errorf("%d: expected 'not found' == %v for err = %v", i, e.notFound, err)
		}
		if fault.IsErrProcess(err) != e.process {
			t.Errorf("%d: expected 'process' == %v for err = %v", i, e.process, err)
		}
		if fault.IsErrRecord(err) != e.record {
			t.Errorf("%d: expected 'record' for err = %v", i, err)
		}
	}
}
