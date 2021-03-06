// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Code generated by MockGen. DO NOT EDIT.
// Source: ../zmqutil/client.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	zmq4 "github.com/pebbe/zmq4"

	util "github.com/bitmark-inc/bitmarkd/util"
	zmqutil "github.com/bitmark-inc/bitmarkd/zmqutil"
)

// MockClient is a mock of Client interface
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Close mocks base method
func (m *MockClient) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockClientMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockClient)(nil).Close))
}

// Connect mocks base method
func (m *MockClient) Connect(conn *util.Connection, serverPublicKey []byte, prefix string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect", conn, serverPublicKey, prefix)
	ret0, _ := ret[0].(error)
	return ret0
}

// Connect indicates an expected call of Connect
func (mr *MockClientMockRecorder) Connect(conn, serverPublicKey, prefix interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockClient)(nil).Connect), conn, serverPublicKey, prefix)
}

// ConnectedTo mocks base method
func (m *MockClient) ConnectedTo() *zmqutil.Connected {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConnectedTo")
	ret0, _ := ret[0].(*zmqutil.Connected)
	return ret0
}

// ConnectedTo indicates an expected call of ConnectedTo
func (mr *MockClientMockRecorder) ConnectedTo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConnectedTo", reflect.TypeOf((*MockClient)(nil).ConnectedTo))
}

// GoString mocks base method
func (m *MockClient) GoString() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GoString")
	ret0, _ := ret[0].(string)
	return ret0
}

// GoString indicates an expected call of GoString
func (mr *MockClientMockRecorder) GoString() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GoString", reflect.TypeOf((*MockClient)(nil).GoString))
}

// IsConnected mocks base method
func (m *MockClient) IsConnected() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsConnected")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsConnected indicates an expected call of IsConnected
func (mr *MockClientMockRecorder) IsConnected() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsConnected", reflect.TypeOf((*MockClient)(nil).IsConnected))
}

// IsConnectedTo mocks base method
func (m *MockClient) IsConnectedTo(serverPublicKey []byte) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsConnectedTo", serverPublicKey)
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsConnectedTo indicates an expected call of IsConnectedTo
func (mr *MockClientMockRecorder) IsConnectedTo(serverPublicKey interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsConnectedTo", reflect.TypeOf((*MockClient)(nil).IsConnectedTo), serverPublicKey)
}

// Reconnect mocks base method
func (m *MockClient) Reconnect() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Reconnect")
	ret0, _ := ret[0].(error)
	return ret0
}

// Reconnect indicates an expected call of Reconnect
func (mr *MockClientMockRecorder) Reconnect() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reconnect", reflect.TypeOf((*MockClient)(nil).Reconnect))
}

// Receive mocks base method
func (m *MockClient) Receive(flags zmq4.Flag) ([][]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Receive", flags)
	ret0, _ := ret[0].([][]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Receive indicates an expected call of Receive
func (mr *MockClientMockRecorder) Receive(flags interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Receive", reflect.TypeOf((*MockClient)(nil).Receive), flags)
}

// Send mocks base method
func (m *MockClient) Send(items ...interface{}) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range items {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Send", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send
func (mr *MockClientMockRecorder) Send(items ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockClient)(nil).Send), items...)
}

// ServerPublicKey mocks base method
func (m *MockClient) ServerPublicKey() []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ServerPublicKey")
	ret0, _ := ret[0].([]byte)
	return ret0
}

// ServerPublicKey indicates an expected call of ServerPublicKey
func (mr *MockClientMockRecorder) ServerPublicKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ServerPublicKey", reflect.TypeOf((*MockClient)(nil).ServerPublicKey))
}

// String mocks base method
func (m *MockClient) String() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String")
	ret0, _ := ret[0].(string)
	return ret0
}

// String indicates an expected call of String
func (mr *MockClientMockRecorder) String() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockClient)(nil).String))
}
