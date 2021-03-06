// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Code generated by MockGen. DO NOT EDIT.
// Source: ../announce/announce.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	fingerprint "github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	rpc "github.com/bitmark-inc/bitmarkd/announce/rpc"
)

// MockAnnounce is a mock of Announce interface
type MockAnnounce struct {
	ctrl     *gomock.Controller
	recorder *MockAnnounceMockRecorder
}

// MockAnnounceMockRecorder is the mock recorder for MockAnnounce
type MockAnnounceMockRecorder struct {
	mock *MockAnnounce
}

// NewMockAnnounce creates a new mock instance
func NewMockAnnounce(ctrl *gomock.Controller) *MockAnnounce {
	mock := &MockAnnounce{ctrl: ctrl}
	mock.recorder = &MockAnnounceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAnnounce) EXPECT() *MockAnnounceMockRecorder {
	return m.recorder
}

// Set mocks base method
func (m *MockAnnounce) Set(arg0 fingerprint.Fingerprint, arg1 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Set", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Set indicates an expected call of Set
func (mr *MockAnnounceMockRecorder) Set(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Set", reflect.TypeOf((*MockAnnounce)(nil).Set), arg0, arg1)
}

// Fetch mocks base method
func (m *MockAnnounce) Fetch(arg0 uint64, arg1 int) ([]rpc.Entry, uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Fetch", arg0, arg1)
	ret0, _ := ret[0].([]rpc.Entry)
	ret1, _ := ret[1].(uint64)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Fetch indicates an expected call of Fetch
func (mr *MockAnnounceMockRecorder) Fetch(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Fetch", reflect.TypeOf((*MockAnnounce)(nil).Fetch), arg0, arg1)
}
