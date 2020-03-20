// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	announce2 "github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/rpc/fixtures"
	"github.com/bitmark-inc/bitmarkd/rpc/listeners"
	"github.com/bitmark-inc/bitmarkd/rpc/mocks"
)

func TestInitialise(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	ann := mocks.NewMockAnnounce(ctl)

	wd, _ := os.Getwd()
	fixtureDir := path.Join(wd, "fixtures")
	cer := fixtures.Certificate(fixtureDir)
	key := fixtures.Key(fixtureDir)

	port := 30000 + rand.Intn(30000)
	listen := fmt.Sprintf("127.0.0.1:%d", port)

	rpcConfig := listeners.RPCConfiguration{
		MaximumConnections: 100,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        cer,
		PrivateKey:         key,
		Announce:           []string{"127.0.0.1:65500"},
	}

	httpsConfig := listeners.HTTPSConfiguration{
		MaximumConnections: 100,
		Listen:             []string{listen},
		Certificate:        cer,
		PrivateKey:         key,
		Allow:              nil,
	}

	ann.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := rpc.Initialise(&rpcConfig, &httpsConfig, "1.0", ann)
	assert.Nil(t, err, "wrong Initialise")

	err = rpc.Finalise()
	assert.Nil(t, err, "wrong Finalise")
}

func TestInitialiseWhenTwice(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	ann := mocks.NewMockAnnounce(ctl)

	wd, _ := os.Getwd()
	fixtureDir := path.Join(wd, "fixtures")
	cer := fixtures.Certificate(fixtureDir)
	key := fixtures.Key(fixtureDir)

	port := 30000 + rand.Intn(30000)
	listen := fmt.Sprintf("127.0.0.1:%d", port)

	rpcConfig := listeners.RPCConfiguration{
		MaximumConnections: 100,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        cer,
		PrivateKey:         key,
		Announce:           []string{"127.0.0.1:65500"},
	}

	httpsConfig := listeners.HTTPSConfiguration{
		MaximumConnections: 100,
		Listen:             []string{listen},
		Certificate:        cer,
		PrivateKey:         key,
		Allow:              nil,
	}

	ann.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := rpc.Initialise(&rpcConfig, &httpsConfig, "1.0", ann)
	assert.Nil(t, err, "wrong Initialise")
	defer rpc.Finalise()

	err = rpc.Initialise(&rpcConfig, &httpsConfig, "1.0", announce2.Get())
	assert.NotNil(t, err, "wrong Initialise")
	assert.Equal(t, fault.AlreadyInitialised, err, "wrong second Initialise")
}

func TestInitialiseWhenCertificateError(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	ann := mocks.NewMockAnnounce(ctl)

	port := 30000 + rand.Intn(30000)
	listen := fmt.Sprintf("127.0.0.1:%d", port)

	rpcConfig := listeners.RPCConfiguration{
		MaximumConnections: 100,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        "",
		PrivateKey:         "",
		Announce:           []string{"127.0.0.1:65500"},
	}

	httpsConfig := listeners.HTTPSConfiguration{}

	err := rpc.Initialise(&rpcConfig, &httpsConfig, "1.0", ann)

	assert.NotNil(t, err, "wrong Initialise")
	assert.Contains(t, err.Error(), "tls", "wrong error")
}

func TestInitialiseWhenRPCListenerError(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	ann := mocks.NewMockAnnounce(ctl)

	wd, _ := os.Getwd()
	fixtureDir := path.Join(wd, "fixtures")
	cer := fixtures.Certificate(fixtureDir)
	key := fixtures.Key(fixtureDir)

	port := 30000 + rand.Intn(30000)
	listen := fmt.Sprintf("127.0.0.1:%d", port)

	rpcConfig := listeners.RPCConfiguration{
		MaximumConnections: 100,
		Bandwidth:          100,
		Listen:             []string{listen},
		Certificate:        cer,
		PrivateKey:         key,
		Announce:           []string{"127.0.0.1:65500"},
	}

	httpsConfig := listeners.HTTPSConfiguration{}

	err := rpc.Initialise(&rpcConfig, &httpsConfig, "1.0", ann)
	assert.NotNil(t, err, "wrong Initialise")
	assert.Equal(t, fault.MissingParameters, err, "wrong error")
}

func TestInitialiseWhenHTTPSListenerError(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	ann := mocks.NewMockAnnounce(ctl)

	wd, _ := os.Getwd()
	fixtureDir := path.Join(wd, "fixtures")
	cer := fixtures.Certificate(fixtureDir)
	key := fixtures.Key(fixtureDir)

	port := 30000 + rand.Intn(30000)
	listen := fmt.Sprintf("127.0.0.1:%d", port)

	rpcConfig := listeners.RPCConfiguration{
		MaximumConnections: 100,
		Bandwidth:          10000000,
		Listen:             []string{listen},
		Certificate:        cer,
		PrivateKey:         key,
		Announce:           []string{"127.0.0.1:65500"},
	}

	httpsConfig := listeners.HTTPSConfiguration{
		MaximumConnections: 0,
		Listen:             []string{listen},
		Certificate:        cer,
		PrivateKey:         key,
		Allow:              nil,
	}

	ann.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := rpc.Initialise(&rpcConfig, &httpsConfig, "1.0", ann)
	assert.NotNil(t, err, "wrong Initialise")
	assert.Equal(t, fault.MissingParameters, err, "wrong error")
}

func TestFinaliseWhenNotInitialised(t *testing.T) {
	err := rpc.Finalise()
	assert.NotNil(t, err, "wrong Finalise")
	assert.Equal(t, fault.NotInitialised, err, "wrong error")
}
