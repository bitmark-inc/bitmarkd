// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package broadcast_test

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	p2pPeer "github.com/libp2p/go-libp2p-core/peer"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/announce/id"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/avl"

	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/bitmarkd/announce/parameter"
	"github.com/golang/mock/gomock"

	"github.com/bitmark-inc/bitmarkd/background"

	"github.com/bitmark-inc/bitmarkd/messagebus"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"

	"github.com/bitmark-inc/bitmarkd/announce/rpc"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/announce/broadcast"
)

const (
	dir         = "testing"
	logCategory = "testing"
)

func setupTestLogger() {
	removeFiles()
	_ = os.Mkdir(dir, 0700)

	logging := logger.Configuration{
		Directory: dir,
		File:      fmt.Sprintf("%s.log", logCategory),
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)
}

func teardownTestLogger() {
	logger.Finalise()
	removeFiles()
}

func removeFiles() {
	err := os.RemoveAll(dir)
	if nil != err {
		fmt.Println("remove dir with error: ", err)
	}
}

func TestRunWhenSendingShutdown(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	log := logger.New(logCategory)
	b := broadcast.New(log, receptor.New(log), rpc.New(), fingerprint.Type{1, 2, 3, 4}, broadcast.UsePeers, parameter.InitialiseInterval, parameter.PollingInterval)

	ch := make(chan messagebus.Message)
	shutdown := make(chan struct{})
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, b background.Process, wg *sync.WaitGroup, sh <-chan struct{}) {
		b.Run(ch, sh)
		wg.Done()
	}(ch, b, wg, shutdown)

	shutdown <- struct{}{}
	wg.Wait()
}

func TestRunWhenConnectedNodeLessThanMinimum(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)
	f := fingerprint.Type{1, 2, 3}
	tree := avl.New()

	rpcs.EXPECT().IsSet().Return(false).Times(1)
	receptors.EXPECT().IsSet().Return(false).Times(1)
	rpcs.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Tree().Return(tree).Times(2)
	receptors.EXPECT().Self().Return(nil).Times(1)
	receptors.EXPECT().Change(false).Return().Times(1)

	b := broadcast.New(
		logger.New(logCategory),
		receptors,
		rpcs,
		f,
		broadcast.UsePeers,
		time.Millisecond,
		time.Minute,
	)

	shutdown := make(chan struct{})
	ch := make(chan messagebus.Message)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, shutdown <-chan struct{}, wg *sync.WaitGroup) {
		b.Run(ch, shutdown)
		wg.Done()
	}(ch, shutdown, wg)

	time.Sleep(5 * time.Millisecond)
	shutdown <- struct{}{}
	wg.Wait()
}

func TestRunWhenConnectedNodeLessThanMinimumThenExhaustive(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)
	f := fingerprint.Type{1, 2, 3}
	now := time.Now()
	tree := avl.New()
	id1 := id.ID("id1")
	data1 := receptor.Data{
		ID:        p2pPeer.ID("id1"),
		Listeners: []ma.Multiaddr{},
		Timestamp: now,
	}
	id2 := id.ID("id2")
	data2 := receptor.Data{
		ID:        p2pPeer.ID("id2"),
		Listeners: []ma.Multiaddr{},
		Timestamp: now,
	}
	id3 := id.ID("id3")
	data3 := receptor.Data{
		ID:        p2pPeer.ID("id3"),
		Listeners: []ma.Multiaddr{},
		Timestamp: now,
	}
	tree.Insert(id1, &data1)
	tree.Insert(id2, &data2)
	tree.Insert(id3, &data3)
	self, _ := tree.Search(id.ID("id1"))

	rpcs.EXPECT().IsSet().Return(false).Times(1)
	receptors.EXPECT().IsSet().Return(false).Times(1)
	rpcs.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Tree().Return(tree).Times(2)
	receptors.EXPECT().Self().Return(self).Times(1)
	receptors.EXPECT().ID().Return(p2pPeer.ID("")).Times(3)
	receptors.EXPECT().Change(false).Return().Times(1)

	b := broadcast.New(
		logger.New(logCategory),
		receptors,
		rpcs,
		f,
		broadcast.UsePeers,
		time.Millisecond,
		time.Minute,
	)

	shutdown := make(chan struct{})
	ch := make(chan messagebus.Message)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, shutdown <-chan struct{}, wg *sync.WaitGroup) {
		b.Run(ch, shutdown)
		wg.Done()
	}(ch, shutdown, wg)

	bus := messagebus.Bus.P2P.Chan()
	received := <-bus
	assert.Equal(t, "ES", received.Command, "wrong command")
	received = <-bus
	assert.Equal(t, "ES", received.Command, "wrong command")
	received = <-bus
	assert.Equal(t, "ES", received.Command, "wrong command")

	time.Sleep(5 * time.Millisecond)
	shutdown <- struct{}{}
	wg.Wait()
}

func TestRunWhenEnoughConnectNode(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)
	f := fingerprint.Type{1, 2, 3}
	tree := avl.New()
	id1 := id.ID("id1")
	id2 := id.ID("id2")
	id3 := id.ID("id3")
	id4 := id.ID("id4")
	id5 := id.ID("id5")
	id6 := id.ID("id6")
	id7 := id.ID("id7")
	tree.Insert(id1, "key1")
	tree.Insert(id2, "key2")
	tree.Insert(id3, "key3")
	tree.Insert(id4, "key4")
	tree.Insert(id5, "key5")
	tree.Insert(id6, "key6")
	tree.Insert(id7, "key7")

	rpcs.EXPECT().IsSet().Return(false).Times(1)
	receptors.EXPECT().IsSet().Return(false).Times(1)
	rpcs.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Tree().Return(tree).Times(1)
	receptors.EXPECT().ReBalance().Return().Times(1)
	receptors.EXPECT().Change(false).Return().Times(1)

	b := broadcast.New(
		logger.New(logCategory),
		receptors,
		rpcs,
		f,
		broadcast.UsePeers,
		time.Millisecond,
		time.Minute,
	)

	shutdown := make(chan struct{})
	ch := make(chan messagebus.Message)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, shutdown <-chan struct{}, wg *sync.WaitGroup) {
		b.Run(ch, shutdown)
		wg.Done()
	}(ch, shutdown, wg)

	time.Sleep(5 * time.Millisecond)
	shutdown <- struct{}{}
	wg.Wait()
}

func TestRunWhenConnectRPCIsSet(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)
	f := fingerprint.Type{1, 2, 3}
	tree := avl.New()
	id1 := id.ID("id1")
	id2 := id.ID("id2")
	id3 := id.ID("id3")
	id4 := id.ID("id4")
	id5 := id.ID("id5")
	id6 := id.ID("id6")
	id7 := id.ID("id7")
	tree.Insert(id1, "key1")
	tree.Insert(id2, "key2")
	tree.Insert(id3, "key3")
	tree.Insert(id4, "key4")
	tree.Insert(id5, "key5")
	tree.Insert(id6, "key6")
	tree.Insert(id7, "key7")

	rpcs.EXPECT().IsSet().Return(true).Times(1)
	receptors.EXPECT().IsSet().Return(false).Times(1)
	rpcs.EXPECT().Self().Return([]byte{}).Times(1)
	rpcs.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Tree().Return(tree).Times(1)
	receptors.EXPECT().ReBalance().Return().Times(1)
	receptors.EXPECT().Change(false).Return().Times(1)

	b := broadcast.New(
		logger.New(logCategory),
		receptors,
		rpcs,
		f,
		broadcast.UsePeers,
		time.Millisecond,
		time.Minute,
	)

	shutdown := make(chan struct{})
	ch := make(chan messagebus.Message)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, shutdown <-chan struct{}, wg *sync.WaitGroup) {
		b.Run(ch, shutdown)
		wg.Done()
	}(ch, shutdown, wg)

	time.Sleep(5 * time.Millisecond)
	bus := messagebus.Bus.P2P.Chan()

	received := <-bus
	assert.Equal(t, "rpc", received.Command, "wrong command")
	shutdown <- struct{}{}
	wg.Wait()
}

func TestRunWhenConnectReceptorIsSet(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	defer messagebus.Bus.P2P.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)
	f := fingerprint.Type{1, 2, 3}
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	tree := avl.New()
	id1 := id.ID("id1")
	id2 := id.ID("id2")
	id3 := id.ID("id3")
	id4 := id.ID("id4")
	id5 := id.ID("id5")
	id6 := id.ID("id6")
	id7 := id.ID("id7")
	tree.Insert(id1, "key1")
	tree.Insert(id2, "key2")
	tree.Insert(id3, "key3")
	tree.Insert(id4, "key4")
	tree.Insert(id5, "key5")
	tree.Insert(id6, "key6")
	tree.Insert(id7, "key7")

	rpcs.EXPECT().IsSet().Return(false).Times(1)
	receptors.EXPECT().IsSet().Return(true).Times(1)
	receptors.EXPECT().SelfAddress().Return([]ma.Multiaddr{addr}).Times(2)
	receptors.EXPECT().ShortID().Return("test").Times(1)
	receptors.EXPECT().BinaryID().Return([]byte{}).Times(1)
	rpcs.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Tree().Return(tree).Times(1)
	receptors.EXPECT().ReBalance().Return().Times(1)
	receptors.EXPECT().Change(false).Return().Times(1)

	b := broadcast.New(
		logger.New(logCategory),
		receptors,
		rpcs,
		f,
		broadcast.UsePeers,
		time.Millisecond,
		time.Minute,
	)

	shutdown := make(chan struct{})
	ch := make(chan messagebus.Message)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, shutdown <-chan struct{}, wg *sync.WaitGroup) {
		b.Run(ch, shutdown)
		wg.Done()
	}(ch, shutdown, wg)

	time.Sleep(5 * time.Millisecond)
	bus := messagebus.Bus.P2P.Chan()

	received := <-bus
	assert.Equal(t, "peer", received.Command, "wrong command")
	shutdown <- struct{}{}
	wg.Wait()
}
