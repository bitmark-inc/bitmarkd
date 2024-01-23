// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package listeners_test

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc/certificate"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/listeners"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/bitmark-inc/logger"
)

type Add struct{}
type AddArg struct {
	A, B int
}

func (a Add) Add(arg *AddArg, reply *int) error {
	*reply = arg.A + arg.B
	return nil
}

func TestRpcListenerServe(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port := rand.Intn(30000) + 30000
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	con := listeners.RPCConfiguration{
		MaximumConnections: 5,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()
	err := s.Register(Add{})
	if err != nil {
		t.Error("register with error: ", err)
		t.FailNow()
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)
	a.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	wd, _ := os.Getwd()
	fixturePath := path.Join(filepath.Dir(wd), "fixtures")
	tlsCertificate, fin, err := certificate.Get(
		logger.New(fixtures.LogCategory),
		"test",
		fixtures.Certificate(fixturePath),
		fixtures.Key(fixturePath),
	)
	if err != nil {
		fmt.Printf("get certificate with error: %s\n", err)
	}

	l, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		tlsCertificate,
		fin,
	)
	assert.Nil(t, err, "wrong NewRPC")

	err = l.Serve()
	assert.Nil(t, err, "wrong Serve")

	tlsConfig := tls.Config{
		InsecureSkipVerify: true,
	}

	c, err := tls.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port), &tlsConfig)
	if err != nil {
		t.Error("dial with error: ", err)
		t.FailNow()
	}

	arg := AddArg{
		A: 2,
		B: 5,
	}
	var reply int

	client := jsonrpc.NewClient(c)
	err = client.Call("Add.Add", &arg, &reply)
	assert.Nil(t, err, "wrong client Call")
	assert.Equal(t, arg.A+arg.B, reply, "wrong result")
}

func TestRpcListenerServeWhenMaxConnectionCountTooSmall(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port := rand.Intn(30000) + 30000
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	con := listeners.RPCConfiguration{
		MaximumConnections: 0,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.Equal(t, fault.MissingParameters, err, "wrong error")
}

func TestRpcListenerServeWhenBandwidthTooSmall(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port := rand.Intn(30000) + 30000
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	con := listeners.RPCConfiguration{
		MaximumConnections: 1,
		Bandwidth:          100,
		Listen:             []string{listen},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.Equal(t, fault.MissingParameters, err, "wrong error")
}

func TestRpcListenerServeWhenEmptyListen(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	con := listeners.RPCConfiguration{
		MaximumConnections: 1,
		Bandwidth:          10000000,
		Listen:             []string{},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.Equal(t, fault.MissingParameters, err, "wrong error")
}

func TestRpcListenerServeWhenWrongAnnounce(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port := rand.Intn(30000) + 30000
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	con := listeners.RPCConfiguration{
		MaximumConnections: 1,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        "",
		Announce:           []string{"", "1"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.NotNil(t, err, "wrong error")
}

func TestRpcListenerServeWhenErrorAnnounceSet(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port := rand.Intn(30000) + 30000
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	con := listeners.RPCConfiguration{
		MaximumConnections: 5,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)
	a.EXPECT().Set(gomock.Any(), gomock.Any()).Return(fmt.Errorf("fake error")).Times(1)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.NotNil(t, err, "wrong error")
	assert.Equal(t, "fake error", err.Error(), "wrong error message")
}

func TestRpcListenerServeWhenErrorListen(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	con := listeners.RPCConfiguration{
		MaximumConnections: 5,
		Bandwidth:          10000000,
		Listen:             []string{"1"},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)
	a.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.NotNil(t, err, "wrong error")
	assert.Equal(t, fault.InvalidIpAddress, err, "wrong error message")
}

func TestRpcListenerServeWhenListenAll(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	con := listeners.RPCConfiguration{
		MaximumConnections: 5,
		Bandwidth:          10000000,
		Listen:             []string{"*:1234"},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)
	a.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.Nil(t, err, "wrong NewRPC")
}

func TestRpcListenerServeWhenListenIPv6(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	con := listeners.RPCConfiguration{
		MaximumConnections: 5,
		Bandwidth:          10000000,
		Listen:             []string{"[1:2:3:4:5:6:7:8]::1"},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)
	a.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.Nil(t, err, "wrong NewRPC")
}

func TestRpcListenerServeWhenInvalidTLSConfig(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	port := rand.Intn(30000) + 30000
	listen := fmt.Sprintf("127.0.0.1:%d", port)
	con := listeners.RPCConfiguration{
		MaximumConnections: 5,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        "",
		Announce:           []string{"127.0.0.1:9999"},
	}

	count := counter.Counter(0)

	s := rpc.NewServer()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	a := mocks.NewMockAnnounce(ctl)
	a.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	l, err := listeners.NewRPC(
		&con,
		logger.New(fixtures.LogCategory),
		&count,
		s,
		a,
		&tls.Config{},
		[32]byte{},
	)
	assert.Nil(t, err, "wrong NewRPC")

	err = l.Serve()
	assert.NotNil(t, err, "wrong Serve")
	assert.Contains(t, err.Error(), "tls", "wrong error message")
}
