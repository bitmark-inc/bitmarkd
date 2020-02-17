// Code generated by MockGen. DO NOT EDIT.
// Source: voting/voting.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	blockdigest "github.com/bitmark-inc/bitmarkd/blockdigest"
	gomock "github.com/golang/mock/gomock"
)

// MockCandidate is a mock of Candidate interface
type MockCandidate struct {
	ctrl     *gomock.Controller
	recorder *MockCandidateMockRecorder
}

// MockCandidateMockRecorder is the mock recorder for MockCandidate
type MockCandidateMockRecorder struct {
	mock *MockCandidate
}

// NewMockCandidate creates a new mock instance
func NewMockCandidate(ctrl *gomock.Controller) *MockCandidate {
	mock := &MockCandidate{ctrl: ctrl}
	mock.recorder = &MockCandidateMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockCandidate) EXPECT() *MockCandidateMockRecorder {
	return m.recorder
}

// CachedRemoteHeight mocks base method
func (m *MockCandidate) CachedRemoteHeight() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CachedRemoteHeight")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// CachedRemoteHeight indicates an expected call of CachedRemoteHeight
func (mr *MockCandidateMockRecorder) CachedRemoteHeight() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CachedRemoteHeight", reflect.TypeOf((*MockCandidate)(nil).CachedRemoteHeight))
}

// CachedRemoteDigestOfLocalHeight mocks base method
func (m *MockCandidate) CachedRemoteDigestOfLocalHeight() blockdigest.Digest {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CachedRemoteDigestOfLocalHeight")
	ret0, _ := ret[0].(blockdigest.Digest)
	return ret0
}

// CachedRemoteDigestOfLocalHeight indicates an expected call of CachedRemoteDigestOfLocalHeight
func (mr *MockCandidateMockRecorder) CachedRemoteDigestOfLocalHeight() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CachedRemoteDigestOfLocalHeight", reflect.TypeOf((*MockCandidate)(nil).CachedRemoteDigestOfLocalHeight))
}

// RemoteAddr mocks base method
func (m *MockCandidate) RemoteAddr() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoteAddr")
	ret0, _ := ret[0].(string)
	return ret0
}

// RemoteAddr indicates an expected call of RemoteAddr
func (mr *MockCandidateMockRecorder) RemoteAddr() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoteAddr", reflect.TypeOf((*MockCandidate)(nil).RemoteAddr))
}

// Name mocks base method
func (m *MockCandidate) Name() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Name")
	ret0, _ := ret[0].(string)
	return ret0
}

// Name indicates an expected call of Name
func (mr *MockCandidateMockRecorder) Name() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Name", reflect.TypeOf((*MockCandidate)(nil).Name))
}
