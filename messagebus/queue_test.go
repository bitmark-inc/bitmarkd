// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messagebus_test

import (
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/messagebus"
)

func setup(t *testing.T) {
	t.Logf("running %s\n", t.Name())
}

func teardown(t *testing.T) {
	messagebus.Bus.Announce.Release()
	messagebus.Bus.Blockstore.Release()
	messagebus.Bus.Connector.Release()
	messagebus.Bus.TestQueue.Release()
	messagebus.Bus.Broadcast.Release()

	messagebus.Bus.Broadcast.DropCache()

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

func TestCache(t *testing.T) {

	setup(t)
	defer teardown(t)

	cacheableCmd := []string{"assets", "issues", "transfer", "proof", "block"}
	uncacheableCmd := []string{"rpc", "peer"}
	c1 := cacheableCmd[rand.Intn(len(cacheableCmd))]
	c2 := uncacheableCmd[rand.Intn(len(uncacheableCmd))]
	c := []string{c1, c2}
	p := []byte{0x05, 0xFA, 0xFE}

	// declare listener
	queue := messagebus.Bus.Broadcast.Chan(50)

	// send a message has not been cached before
	for _, cmd := range c {
		messagebus.Bus.Broadcast.Send(cmd, p)
	}

	// wait for background
	time.Sleep(20 * time.Millisecond)

	s := make([]string, 0)
	for len(queue) > 0 {
		m := <-queue
		s = append(s, m.Command)
	}

	// verify received values from queue are the ones sent before
	if len(c) != len(s) {
		t.Errorf("actual count: %d, expected: %d", len(s), len(c))
	}

	for i := 0; i < len(c); i += 1 {
		if c[i] != s[i] {
			t.Errorf("actual command: %s, expected: %s", s[i], c[i])
		}
	}

	// func to check whether a string is contained in an array string
	f := func(a []string, i string) bool {
		for _, item := range a {
			if item == i {
				return true
			}
		}
		return false
	}

	// recreate s
	s = make([]string, 0)
	for _, cmd := range c {
		messagebus.Bus.Broadcast.Send(cmd, p)
	}
	// wait for background
	time.Sleep(20 * time.Millisecond)

	for len(queue) > 0 {
		m := <-queue
		s = append(s, m.Command)
	}

	// verify the queue did not contains cache value
	if len(c) == len(s) {
		t.Errorf("actual count: %d, expected: %d", len(s), len(c))
	}

	for i := 0; i < len(s); i += 1 {
		if !f(uncacheableCmd, s[i]) {
			t.Errorf("actual: %q, expected in %q", s[i], uncacheableCmd)
		}
	}

	// drop cache and resend it
	params := [][]byte{p}
	messagebus.Bus.Broadcast.DropCache(messagebus.Message{Command: c1, Parameters: params})
	messagebus.Bus.Broadcast.Send(c1, p)
	// wait for background
	time.Sleep(20 * time.Millisecond)

	select {
	case received := <-queue:
		if received.Command != c1 {
			t.Errorf("actual command: %q, expected: %q", received.Command, c1)
		}
	default:
		t.Errorf("actual nothing but expected is %q", c1)
	}

}

func TestQueueOverflow(t *testing.T) {

	setup(t)
	defer teardown(t)

	const queueSize = 10
	cmd := []string{"assets", "issues", "transfer", "proof", "rpc", "peer"}
	p1 := []byte{0x11, 0x99}
	p2 := []byte{0xAA, 0xFF}
	queue := messagebus.Bus.Broadcast.Chan(queueSize)

	// put at least 1 block message
	messagebus.Bus.Broadcast.Send("block", p1)
	for i := 0; i < queueSize-1; i += 1 {
		c := cmd[rand.Intn(len(cmd))]
		p := make([]byte, rand.Intn(1024))
		rand.Read(p)
		messagebus.Bus.Broadcast.Send(c, p)
	}

	time.Sleep(20 * time.Millisecond)

	// verify the queue is full
	if queueSize != len(queue) {
		t.Errorf("len(queue) actual: %d, expected: %d", len(queue), queueSize)
	}

	// put one more block message into queue
	messagebus.Bus.Broadcast.Send("block", p2)
	time.Sleep(20 * time.Millisecond)

	// verify there are 2 block messages in queue
	count := 0
	for len(queue) > 0 {
		m := <-queue
		if "block" == m.Command {
			count += 1
		}
	}
	if 2 != count {
		t.Errorf("block message count actual: %d, expected: %d", count, 2)
	}

	// make the queue is full
	for i := 0; i < queueSize; i += 1 {
		c := cmd[rand.Intn(len(cmd))]
		p := make([]byte, rand.Intn(1024))
		rand.Read(p)
		messagebus.Bus.Broadcast.Send(c, p)
	}

	time.Sleep(20 * time.Millisecond)

	// verify the queue is full
	if queueSize != len(queue) {
		t.Errorf("len(queue) actual: %d, expected: %d", len(queue), queueSize)
	}

	// put one non-block message to the queue
	c := cmd[rand.Intn(len(cmd))]
	messagebus.Bus.Broadcast.Send(c, p2)
	time.Sleep(20 * time.Millisecond)

	// verify the new message is just dropped
	for len(queue) > 0 {
		m := <-queue
		if c == m.Command && 1 == len(m.Parameters) && reflect.DeepEqual(p2, m.Parameters[0]) {
			t.Error("Expected new message won't be in queue but actually it's")
		}
	}

}

func TestCacheStateQueueOverflow(t *testing.T) {

	setup(t)
	defer teardown(t)

	const queueSize = 3
	cmd := []string{"assets", "issues", "transfer", "proof", "rpc", "peer"}

	// listener
	queue := messagebus.Bus.Broadcast.Chan(queueSize)

	// make the queue is overflow and continue send one item
	for i := 0; i < queueSize; i++ {
		c := cmd[rand.Intn(len(cmd))]
		p := make([]byte, rand.Intn(1024))
		rand.Read(p)
		messagebus.Bus.Broadcast.Send(c, p)
	}

	p := []byte{0x0A, 0x9F}
	c := cmd[0]

	// continue to send one more
	messagebus.Bus.Broadcast.Send(c, p)

	time.Sleep(20 * time.Millisecond)

	// drop one item
	<-queue

	// then push the last sent item
	messagebus.Bus.Broadcast.Send(c, p)

	time.Sleep(20 * time.Millisecond)

	// verify it already contains in queue
	found := false

searchLoop:
	for len(queue) > 0 {
		m := <-queue
		if c == m.Command && len(m.Parameters) == 1 && reflect.DeepEqual(p, m.Parameters[0]) {
			found = true
			break searchLoop
		}
	}

	if !found {
		t.Errorf("Expected the message command %q is contained in queue, but actually not", c)
	}

}
