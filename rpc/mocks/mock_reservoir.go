// Code generated by MockGen. DO NOT EDIT.
// Source: ../reservoir/setup.go

// Package mocks is a generated GoMock package.
package mocks

import (
	pay "github.com/bitmark-inc/bitmarkd/pay"
	reservoir "github.com/bitmark-inc/bitmarkd/reservoir"
	transactionrecord "github.com/bitmark-inc/bitmarkd/transactionrecord"
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockReservoir is a mock of Reservoir interface
type MockReservoir struct {
	ctrl     *gomock.Controller
	recorder *MockReservoirMockRecorder
}

// MockReservoirMockRecorder is the mock recorder for MockReservoir
type MockReservoirMockRecorder struct {
	mock *MockReservoir
}

// NewMockReservoir creates a new mock instance
func NewMockReservoir(ctrl *gomock.Controller) *MockReservoir {
	mock := &MockReservoir{ctrl: ctrl}
	mock.recorder = &MockReservoirMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockReservoir) EXPECT() *MockReservoirMockRecorder {
	return m.recorder
}

// StoreTransfer mocks base method
func (m *MockReservoir) StoreTransfer(arg0 transactionrecord.BitmarkTransfer) (*reservoir.TransferInfo, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreTransfer", arg0)
	ret0, _ := ret[0].(*reservoir.TransferInfo)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// StoreTransfer indicates an expected call of StoreTransfer
func (mr *MockReservoirMockRecorder) StoreTransfer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreTransfer", reflect.TypeOf((*MockReservoir)(nil).StoreTransfer), arg0)
}

// StoreIssues mocks base method
func (m *MockReservoir) StoreIssues(issues []*transactionrecord.BitmarkIssue) (*reservoir.IssueInfo, bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StoreIssues", issues)
	ret0, _ := ret[0].(*reservoir.IssueInfo)
	ret1, _ := ret[1].(bool)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// StoreIssues indicates an expected call of StoreIssues
func (mr *MockReservoirMockRecorder) StoreIssues(issues interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StoreIssues", reflect.TypeOf((*MockReservoir)(nil).StoreIssues), issues)
}

// TryProof mocks base method
func (m *MockReservoir) TryProof(arg0 pay.PayId, arg1 []byte) reservoir.TrackingStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TryProof", arg0, arg1)
	ret0, _ := ret[0].(reservoir.TrackingStatus)
	return ret0
}

// TryProof indicates an expected call of TryProof
func (mr *MockReservoirMockRecorder) TryProof(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TryProof", reflect.TypeOf((*MockReservoir)(nil).TryProof), arg0, arg1)
}
