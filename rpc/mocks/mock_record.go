// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Code generated by MockGen. DO NOT EDIT.
// Source: ../blockrecord/header.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	blockdigest "github.com/bitmark-inc/bitmarkd/blockdigest"
	blockrecord "github.com/bitmark-inc/bitmarkd/blockrecord"
)

// MockRecord is a mock of Record interface
type MockRecord struct {
	ctrl     *gomock.Controller
	recorder *MockRecordMockRecorder
}

// MockRecordMockRecorder is the mock recorder for MockRecord
type MockRecordMockRecorder struct {
	mock *MockRecord
}

// NewMockRecord creates a new mock instance
func NewMockRecord(ctrl *gomock.Controller) *MockRecord {
	mock := &MockRecord{ctrl: ctrl}
	mock.recorder = &MockRecordMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRecord) EXPECT() *MockRecordMockRecorder {
	return m.recorder
}

// ExtractHeader mocks base method
func (m *MockRecord) ExtractHeader(arg0 []byte, arg1 uint64, arg2 bool) (*blockrecord.Header, blockdigest.Digest, []byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExtractHeader", arg0, arg1, arg2)
	ret0, _ := ret[0].(*blockrecord.Header)
	ret1, _ := ret[1].(blockdigest.Digest)
	ret2, _ := ret[2].([]byte)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// ExtractHeader indicates an expected call of ExtractHeader
func (mr *MockRecordMockRecorder) ExtractHeader(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExtractHeader", reflect.TypeOf((*MockRecord)(nil).ExtractHeader), arg0, arg1, arg2)
}