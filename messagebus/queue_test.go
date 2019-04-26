// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messagebus_test

import (
	"sync"
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/messagebus"
)

func TestQueue(t *testing.T) {

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

	// allow background to run
	time.Sleep(20 * time.Millisecond)

	// create some listeners
	const listeners = 5

	var l [listeners]int
	var wg sync.WaitGroup

	for i := 0; i < listeners; i += 1 {
		wg.Add(1)
		go func(n int) {
			queue := messagebus.Bus.Broadcast.Chan(0)
			for _, item := range items {
				received := <-queue
				if received.Command != item.Command {
					t.Errorf("actual: %q  expected: %q", received.Command, item.Command)
				} else {
					l[n] += 1
				}
			}
			wg.Done()
		}(i)

	}

	// all listening so these messages should be received
	for _, item := range items {
		time.Sleep(20 * time.Millisecond)
		messagebus.Bus.Broadcast.Send(item.Command)
	}
	for _, item := range items {
		time.Sleep(20 * time.Millisecond)
		messagebus.Bus.Broadcast.Send(item.Command)
	}

	// wait for completion
	wg.Wait()
	for i, n := range l {
		if n != len(items) {
			t.Errorf("listener[%d] received: %d  expected: %d", i, n, len(items))
		}
	}
}
