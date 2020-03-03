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

	"github.com/bitmark-inc/bitmarkd/rpc/certificate"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
	"github.com/golang/mock/gomock"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/listeners"
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
	if nil != err {
		fmt.Println("register with error: ", err)
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
	if nil != err {
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

	c, err := tls.Dial("tcp", fmt.Sprintf(":%d", port), &tlsConfig)
	if nil != err {
		fmt.Println("dial with error: ", err)
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
