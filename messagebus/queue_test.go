// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messagebus_test

import (
	"math/rand"
	"sync"
	"testing"

	"github.com/bitmark-inc/bitmarkd/messagebus"
)

// command to exit goroutines
const DONE = "***DONE***"

func setup(t *testing.T) {
	t.Logf("running %s\n", t.Name())
}

func teardown(t *testing.T) {
	messagebus.Bus.Announce.Release()
	messagebus.Bus.Blockstore.Release()
	messagebus.Bus.Connector.Release()
	messagebus.Bus.TestQueue.Release()
	messagebus.Bus.Broadcast.Release()
}

func TestQueue(t *testing.T) {

	setup(t)
	defer teardown(t)

	items := []messagebus.Message{
		{
			Command:    "c1",
			Parameters: nil,
		},
		{
			Command:    "c2",
			Parameters: nil,
		},
		{
			Command:    "c3",
			Parameters: nil,
		},
	}

	for _, item := range items {
		messagebus.Bus.TestQueue.Send(item.Command)
	}

	queue := messagebus.Bus.TestQueue.Chan()
	for _, item := range items {
		received := <-queue
		if received.Command != item.Command {
			t.Errorf("actual: %q  expected: %q", received.Command, item.Command)
		}
	}

}

func TestBroadcast(t *testing.T) {

	setup(t)
	defer teardown(t)

	items := []messagebus.Message{
		{
			Command:    "c1",
			Parameters: nil,
		},
		{
			Command:    "c2",
			Parameters: nil,
		},
		{
			Command:    "c3",
			Parameters: nil,
		},
	}

	// nothing listening so these messages should be dropped
	for _, item := range items {
		messagebus.Bus.Broadcast.Send("ignored:" + item.Command)
	}

	// create some listeners
	const listeners = 5

	var l [listeners]int
	var wgFirst sync.WaitGroup
	var wgStop sync.WaitGroup

	for i := 0; i < listeners; i += 1 {
		wgFirst.Add(1)
		wgStop.Add(1)

		// queue created outside to avoid having spurious sleeps
		// to wait for goroutines to start
		// ensure queue is of sufficient size to prevent deadlock
		queue := messagebus.Bus.Broadcast.Chan(len(items) + 1)

		go func(n int, queue <-chan messagebus.Message) {
		loop:
			for {
				for i, item := range items {
					received := <-queue
					if DONE == received.Command {
						break loop
					}
					if received.Command != item.Command {
						t.Errorf("%d: actual: %q  expected: %q", n, received.Command, item.Command)
					} else {
						l[n] += 1
					}
					if 0 == i {
						wgFirst.Done()
					}
				}
			}
			wgStop.Done()
		}(i, queue)

	}

	// all listening so one copy of each messages should be received
	for _, item := range items {
		for i := 0; i < 10; i += 1 {
			messagebus.Bus.Broadcast.Send(item.Command)
		}
	}

	for _, item := range items {
		messagebus.Bus.Broadcast.Send(item.Command)
	}

	// wait for at least one item removed from all queues
	wgFirst.Wait()

	messagebus.Bus.Broadcast.Send(DONE)

	// wait for final completion
	wgStop.Wait()

	// check right number of items received
	for i, n := range l {
		if n != len(items) {
			t.Errorf("listener[%d] received: %d  expected: %d", i, n, len(items))
		}
	}
}

func TestQueueOverflow(t *testing.T) {

	setup(t)
	defer teardown(t)

	const queueSize = 15

	cmd := []string{"assets", "issues", "transfer", "proof", "rpc", "peer"}

	queue := messagebus.Bus.Broadcast.Chan(queueSize)

	// fill the queue
	for i := 0; i < 2*queueSize; i += 1 {
		c := cmd[rand.Intn(len(cmd))]
		p := make([]byte, rand.Intn(1024))
		rand.Read(p)
		messagebus.Bus.Broadcast.Send(c, p)
	}

	if len(queue) >= queueSize {
		t.Fatal("queue was filled by normal messages")
	}

	messagebus.Bus.Broadcast.Send("block", []byte{0x11, 0x99})

	if len(queue) != queueSize {
		t.Fatal("queue could not accept block")
	}

	// verify block message in queue
	count := 0

	for len(queue) > 0 {
		m := <-queue
		if "block" == m.Command {
			count += 1
		}
	}
	if 0 == count {
		t.Fatal("no block message in queue")
	}
}
