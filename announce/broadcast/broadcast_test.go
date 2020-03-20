// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package broadcast_test

import (
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce/broadcast"
	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	"github.com/bitmark-inc/bitmarkd/announce/fixtures"
	"github.com/bitmark-inc/bitmarkd/announce/id"
	"github.com/bitmark-inc/bitmarkd/announce/mocks"
	"github.com/bitmark-inc/bitmarkd/announce/parameter"
	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/announce/rpc"
	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
)

func TestRunWhenSendingShutdown(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	log := logger.New(fixtures.LogCategory)
	b := broadcast.New(log, receptor.New(log), rpc.New(), parameter.InitialiseInterval, parameter.PollingInterval)

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

func TestRunWhenConnectedReceptorsChanged(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	defer messagebus.Bus.Broadcast.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)

	rpcs.EXPECT().IsSet().Return(false).Times(1)
	rpcs.EXPECT().Expire().Return().Times(1)

	receptors.EXPECT().IsSet().Return(false).Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Changed().Return(true).Times(1)
	receptors.EXPECT().Rebalance().Return().Times(1)
	receptors.EXPECT().Change(false).Return().Times(1)

	b := broadcast.New(
		logger.New(fixtures.LogCategory),
		receptors,
		rpcs,
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

func TestRunWhenRPCIsSet(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()
	defer messagebus.Bus.Broadcast.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)
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
	rpcs.EXPECT().ID().Return(fingerprint.Fingerprint{1, 2, 3, 4}).Times(1)
	rpcs.EXPECT().Self().Return([]byte{}).Times(1)
	rpcs.EXPECT().Expire().Return().Times(1)

	receptors.EXPECT().IsSet().Return(false).Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Changed().Return(false).Times(1)

	b := broadcast.New(
		logger.New(fixtures.LogCategory),
		receptors,
		rpcs,
		time.Millisecond,
		time.Minute,
	)

	bus := messagebus.Bus.Broadcast.Chan(-1)
	shutdown := make(chan struct{})
	ch := make(chan messagebus.Message)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, shutdown <-chan struct{}, wg *sync.WaitGroup) {
		b.Run(ch, shutdown)
		wg.Done()
	}(ch, shutdown, wg)

	time.Sleep(5 * time.Millisecond)

	received := <-bus
	assert.Equal(t, "rpc", received.Command, "wrong command")
	shutdown <- struct{}{}
	wg.Wait()
}

func TestRunWhenReceptorIsSet(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()
	defer messagebus.Bus.Broadcast.Release()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	receptors := mocks.NewMockReceptor(ctl)
	rpcs := mocks.NewMockRPC(ctl)
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
	rpcs.EXPECT().Expire().Return().Times(1)

	receptors.EXPECT().IsSet().Return(true).Times(1)
	receptors.EXPECT().ID().Return(id.ID("test")).Times(2)
	receptors.EXPECT().SelfAddress().Return([]byte{1, 2, 3, 4}).Times(1)
	receptors.EXPECT().Expire().Return().Times(1)
	receptors.EXPECT().Changed().Return(false).Times(1)

	b := broadcast.New(
		logger.New(fixtures.LogCategory),
		receptors,
		rpcs,
		time.Millisecond,
		time.Minute,
	)

	bus := messagebus.Bus.Broadcast.Chan(5)
	shutdown := make(chan struct{})
	ch := make(chan messagebus.Message)
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(ch <-chan messagebus.Message, shutdown <-chan struct{}, wg *sync.WaitGroup) {
		b.Run(ch, shutdown)
		wg.Done()
	}(ch, shutdown, wg)

	time.Sleep(5 * time.Millisecond)

	received := <-bus
	assert.Equal(t, "peer", received.Command, "wrong command")
	shutdown <- struct{}{}
	wg.Wait()
}
