// Code generated by MockGen. DO NOT EDIT.
// Source: receptor/receptor.go

// Package mocks is a generated GoMock package.
package mocks

import (
	avl "github.com/bitmark-inc/bitmarkd/avl"
	gomock "github.com/golang/mock/gomock"
	peer "github.com/libp2p/go-libp2p-core/peer"
	go_multiaddr "github.com/multiformats/go-multiaddr"
	reflect "reflect"
	time "time"
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
func (m *MockReceptor) Add(arg0 peer.ID, arg1 []go_multiaddr.Multiaddr, arg2 uint64) bool {
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

// Changed mocks base method
func (m *MockReceptor) Changed() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Changed")
	ret0, _ := ret[0].(bool)
	return ret0
}

// Changed indicates an expected call of Changed
func (mr *MockReceptorMockRecorder) Changed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Changed", reflect.TypeOf((*MockReceptor)(nil).Changed))
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

// IsSet mocks base method
func (m *MockReceptor) IsSet() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsSet")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsSet indicates an expected call of IsSet
func (mr *MockReceptorMockRecorder) IsSet() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsSet", reflect.TypeOf((*MockReceptor)(nil).IsSet))
}

// Next mocks base method
func (m *MockReceptor) Next(arg0 peer.ID) (peer.ID, []go_multiaddr.Multiaddr, time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next", arg0)
	ret0, _ := ret[0].(peer.ID)
	ret1, _ := ret[1].([]go_multiaddr.Multiaddr)
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
func (m *MockReceptor) Random(arg0 peer.ID) (peer.ID, []go_multiaddr.Multiaddr, time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Random", arg0)
	ret0, _ := ret[0].(peer.ID)
	ret1, _ := ret[1].([]go_multiaddr.Multiaddr)
	ret2, _ := ret[2].(time.Time)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Random indicates an expected call of Random
func (mr *MockReceptorMockRecorder) Random(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Random", reflect.TypeOf((*MockReceptor)(nil).Random), arg0)
}

// SetSelf mocks base method
func (m *MockReceptor) SetSelf(arg0 peer.ID, arg1 []go_multiaddr.Multiaddr) error {
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

// SelfAddress mocks base method
func (m *MockReceptor) SelfAddress() []go_multiaddr.Multiaddr {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SelfAddress")
	ret0, _ := ret[0].([]go_multiaddr.Multiaddr)
	return ret0
}

// SelfAddress indicates an expected call of SelfAddress
func (mr *MockReceptorMockRecorder) SelfAddress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SelfAddress", reflect.TypeOf((*MockReceptor)(nil).SelfAddress))
}

// Tree mocks base method
func (m *MockReceptor) Tree() *avl.Tree {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Tree")
	ret0, _ := ret[0].(*avl.Tree)
	return ret0
}

// Tree indicates an expected call of Tree
func (mr *MockReceptorMockRecorder) Tree() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Tree", reflect.TypeOf((*MockReceptor)(nil).Tree))
}

// ID mocks base method
func (m *MockReceptor) ID() peer.ID {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(peer.ID)
	return ret0
}

// ID indicates an expected call of ID
func (mr *MockReceptorMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockReceptor)(nil).ID))
}

// BinaryID mocks base method
func (m *MockReceptor) BinaryID() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BinaryID")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// BinaryID indicates an expected call of BinaryID
func (mr *MockReceptorMockRecorder) BinaryID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BinaryID", reflect.TypeOf((*MockReceptor)(nil).BinaryID))
}

// ShortID mocks base method
func (m *MockReceptor) ShortID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ShortID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ShortID indicates an expected call of ShortID
func (mr *MockReceptorMockRecorder) ShortID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ShortID", reflect.TypeOf((*MockReceptor)(nil).ShortID))
}

// UpdateTime mocks base method
func (m *MockReceptor) UpdateTime(arg0 peer.ID, arg1 time.Time) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UpdateTime", arg0, arg1)
}

// UpdateTime indicates an expected call of UpdateTime
func (mr *MockReceptorMockRecorder) UpdateTime(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTime", reflect.TypeOf((*MockReceptor)(nil).UpdateTime), arg0, arg1)
}

// ReBalance mocks base method
func (m *MockReceptor) ReBalance() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "ReBalance")
}

// ReBalance indicates an expected call of ReBalance
func (mr *MockReceptorMockRecorder) ReBalance() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ReBalance", reflect.TypeOf((*MockReceptor)(nil).ReBalance))
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
