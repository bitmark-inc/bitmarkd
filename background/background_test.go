// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package background_test

import (
	"github.com/bitmark-inc/bitmarkd/background"
	"testing"
	"time"
)

type bg1 struct {
	count int
}

const (
	initialCount1 = 246
	finalCount1   = 987654321
	initialCount2 = 777
	finalCount2   = 897645312
)

func TestBackground(t *testing.T) {

	proc1 := &bg1{
		count: initialCount1,
	}
	proc2 := &bg1{
		count: initialCount2,
	}

	// list of background processes to start
	processes := background.Processes{
		proc1,
		proc2,
	}

	p := background.Start(processes, t)
	time.Sleep(50 * time.Millisecond)
	p.Stop()
	time.Sleep(20 * time.Millisecond)

	if finalCount1 != proc1.count {
		t.Fatalf("stop failed: final value expected: %d  actual: %d", finalCount1, proc1.count)
	}
	if finalCount2 != proc2.count {
		t.Fatalf("stop failed: final value expected: %d  actual: %d", finalCount2, proc2.count)
	}
}

func (state *bg1) Run(args interface{}, shutdown <-chan struct{}) {

	t := args.(*testing.T)

	n := 0
	if initialCount1 == state.count {
		n = 1
	} else if initialCount2 == state.count {
		n = 2
	} else {
		t.Errorf("initialisation failed: unexpected initial count: %d", state.count)
	}

loop:
	for {
		select {
		case <-shutdown:
			break loop
		default:
		}
		state.count += 9
		t.Logf("state[%d]: %v", n, state)
		time.Sleep(time.Millisecond)
	}

	// test for the stop operation
	switch n {
	case 1:
		state.count = finalCount1
	case 2:
		state.count = finalCount2
	default:
		t.Errorf("unexpected n: %d", n)
	}
}

type bg2 struct {
	count int
}

func TestTimeout(t *testing.T) {

	proc1 := &bg2{
		count: 1,
	}
	proc2 := &bg2{
		count: 2,
	}

	// list of background processes to start
	processes := background.Processes{
		proc1,
		proc2,
	}

	start := time.Now()
	p := background.Start(processes, t)
	time.Sleep(50 * time.Millisecond)
	p.Stop()
	finish := time.Now()
	deltaT := finish.Sub(start)
	if deltaT > 10*time.Second {
		t.Errorf("too long: %s", deltaT)
	} else {
		t.Logf("shutdown time: %s", deltaT)
	}
}

func (state *bg2) Run(args interface{}, shutdown <-chan struct{}) {

	t := args.(*testing.T)

	n := 0
	if state.count > 2 {
		t.Errorf("initialisation failed: unexpected initial count: %d", state.count)
	}

	for {
		state.count += 1
		t.Logf("state[%d]: %v", n, state)
		time.Sleep(time.Second)
	}
}
