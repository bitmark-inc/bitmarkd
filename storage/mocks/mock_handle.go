// Code generated by MockGen. DO NOT EDIT.
// Source: handle.go

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockHandle is a mock of Handle interface
type MockHandle struct {
	ctrl     *gomock.Controller
	recorder *MockHandleMockRecorder
}

// MockHandleMockRecorder is the mock recorder for MockHandle
type MockHandleMockRecorder struct {
	mock *MockHandle
}

// NewMockHandle creates a new mock instance
func NewMockHandle(ctrl *gomock.Controller) *MockHandle {
	mock := &MockHandle{ctrl: ctrl}
	mock.recorder = &MockHandleMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockHandle) EXPECT() *MockHandleMockRecorder {
	return m.recorder
}

// put mocks base method
func (m *MockHandle) put(arg0, arg1, arg2 []byte) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "put", arg0, arg1, arg2)
}

// put indicates an expected call of put
func (mr *MockHandleMockRecorder) put(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "put", reflect.TypeOf((*MockHandle)(nil).put), arg0, arg1, arg2)
}

// putN mocks base method
func (m *MockHandle) putN(arg0 []byte, arg1 uint64) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "putN", arg0, arg1)
}

// putN indicates an expected call of putN
func (mr *MockHandleMockRecorder) putN(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "putN", reflect.TypeOf((*MockHandle)(nil).putN), arg0, arg1)
}

// remove mocks base method
func (m *MockHandle) remove(arg0 []byte) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "remove", arg0)
}

// remove indicates an expected call of remove
func (mr *MockHandleMockRecorder) remove(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "remove", reflect.TypeOf((*MockHandle)(nil).remove), arg0)
}

// Get mocks base method
func (m *MockHandle) Get(arg0 []byte) []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0)
	ret0, _ := ret[0].([]byte)
	return ret0
}

// Get indicates an expected call of Get
func (mr *MockHandleMockRecorder) Get(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockHandle)(nil).Get), arg0)
}

// GetN mocks base method
func (m *MockHandle) GetN(arg0 []byte) (uint64, bool) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetN", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(bool)
	return ret0, ret1
}

// GetN indicates an expected call of GetN
func (mr *MockHandleMockRecorder) GetN(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetN", reflect.TypeOf((*MockHandle)(nil).GetN), arg0)
}

// GetNB mocks base method
func (m *MockHandle) GetNB(arg0 []byte) (uint64, []byte) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNB", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].([]byte)
	return ret0, ret1
}

// GetNB indicates an expected call of GetNB
func (mr *MockHandleMockRecorder) GetNB(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNB", reflect.TypeOf((*MockHandle)(nil).GetNB), arg0)
}

// Has mocks base method
func (m *MockHandle) Has(arg0 []byte) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Has", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Has indicates an expected call of Has
func (mr *MockHandleMockRecorder) Has(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Has", reflect.TypeOf((*MockHandle)(nil).Has), arg0)
}

// Begin mocks base method
func (m *MockHandle) Begin() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Begin")
}

// Begin indicates an expected call of Begin
func (mr *MockHandleMockRecorder) Begin() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Begin", reflect.TypeOf((*MockHandle)(nil).Begin))
}

// Commit mocks base method
func (m *MockHandle) Commit() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Commit")
	ret0, _ := ret[0].(error)
	return ret0
}

// Commit indicates an expected call of Commit
func (mr *MockHandleMockRecorder) Commit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Commit", reflect.TypeOf((*MockHandle)(nil).Commit))
}