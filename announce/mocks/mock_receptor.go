// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Code generated by MockGen. DO NOT EDIT.
// Source: receptor/receptor.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"

	id "github.com/bitmark-inc/bitmarkd/announce/id"
	avl "github.com/bitmark-inc/bitmarkd/avl"
)

// MockReceptor is a mock of Receptor interface
type MockReceptor struct {
	ctrl     *gomock.Controller
	recorder *MockReceptorMockRecorder
}

// MockReceptorMockRecorder is the mock recorder for MockReceptor
type MockReceptorMockRecorder struct {
	mock *MockReceptor
}

// NewMockReceptor creates a new mock instance
func NewMockReceptor(ctrl *gomock.Controller) *MockReceptor {
	mock := &MockReceptor{ctrl: ctrl}
	mock.recorder = &MockReceptorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockReceptor) EXPECT() *MockReceptorMockRecorder {
	return m.recorder
}

// Add mocks base method
func (m *MockReceptor) Add(arg0, arg1 []byte, arg2 uint64) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Add", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Add indicates an expected call of Add
func (mr *MockReceptorMockRecorder) Add(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockReceptor)(nil).Add), arg0, arg1, arg2)
}

// SetSelf mocks base method
func (m *MockReceptor) SetSelf(arg0, arg1 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetSelf", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetSelf indicates an expected call of SetSelf
func (mr *MockReceptorMockRecorder) SetSelf(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSelf", reflect.TypeOf((*MockReceptor)(nil).SetSelf), arg0, arg1)
}

// Next mocks base method
func (m *MockReceptor) Next(arg0 []byte) ([]byte, []byte, time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].([]byte)
	ret2, _ := ret[2].(time.Time)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Next indicates an expected call of Next
func (mr *MockReceptorMockRecorder) Next(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockReceptor)(nil).Next), arg0)
}

// Random mocks base method
func (m *MockReceptor) Random(arg0 []byte) ([]byte, []byte, time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Random", arg0)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].([]byte)
	ret2, _ := ret[2].(time.Time)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Random indicates an expected call of Random
func (mr *MockReceptorMockRecorder) Random(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Random", reflect.TypeOf((*MockReceptor)(nil).Random), arg0)
}

// ReBalance mocks base method
func (m *MockReceptor) ReBalance() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ReBalance")
}

// ReBalance indicates an expected call of ReBalance
func (mr *MockReceptorMockRecorder) Rebalance() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReBalance", reflect.TypeOf((*MockReceptor)(nil).ReBalance))
}

// UpdateTime mocks base method
func (m *MockReceptor) UpdateTime(arg0 []byte, arg1 time.Time) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateTime", arg0, arg1)
}

// UpdateTime indicates an expected call of UpdateTime
func (mr *MockReceptorMockRecorder) UpdateTime(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTime", reflect.TypeOf((*MockReceptor)(nil).UpdateTime), arg0, arg1)
}

// IsChanged mocks base method
func (m *MockReceptor) IsChanged() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsChanged")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsChanged indicates an expected call of IsChanged
func (mr *MockReceptorMockRecorder) Changed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsChanged", reflect.TypeOf((*MockReceptor)(nil).IsChanged))
}

// Change mocks base method
func (m *MockReceptor) Change(arg0 bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Change", arg0)
}

// Change indicates an expected call of Change
func (mr *MockReceptorMockRecorder) Change(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Change", reflect.TypeOf((*MockReceptor)(nil).Change), arg0)
}

// IsInitialised mocks base method
func (m *MockReceptor) IsInitialised() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsInitialised")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsInitialised indicates an expected call of IsInitialised
func (mr *MockReceptorMockRecorder) IsSet() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsInitialised", reflect.TypeOf((*MockReceptor)(nil).IsInitialised))
}

// Connectable mocks base method
func (m *MockReceptor) Connectable() *avl.Tree {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connectable")
	ret0, _ := ret[0].(*avl.Tree)
	return ret0
}

// Connectable indicates an expected call of Connectable
func (mr *MockReceptorMockRecorder) Connectable() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connectable", reflect.TypeOf((*MockReceptor)(nil).Connectable))
}

// ID mocks base method
func (m *MockReceptor) ID() id.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(id.ID)
	return ret0
}

// ID indicates an expected call of ID
func (mr *MockReceptorMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockReceptor)(nil).ID))
}

// Self mocks base method
func (m *MockReceptor) Self() *avl.Node {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Self")
	ret0, _ := ret[0].(*avl.Node)
	return ret0
}

// Self indicates an expected call of Self
func (mr *MockReceptorMockRecorder) Self() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Self", reflect.TypeOf((*MockReceptor)(nil).Self))
}

// SelfListener mocks base method
func (m *MockReceptor) SelfListener() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SelfListener")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// SelfListener indicates an expected call of SelfListener
func (mr *MockReceptorMockRecorder) SelfAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelfListener", reflect.TypeOf((*MockReceptor)(nil).SelfListener))
}

// Expire mocks base method
func (m *MockReceptor) Expire() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Expire")
}

// Expire indicates an expected call of Expire
func (mr *MockReceptorMockRecorder) Expire() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Expire", reflect.TypeOf((*MockReceptor)(nil).Expire))
}
