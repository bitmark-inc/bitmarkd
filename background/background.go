// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package background

import (
	"sync"
)

// the shutdown and completed type for a background
type shutdown struct {
	shutdown chan struct{}
	finished chan struct{}
}

// T - handle type for the stop
type T struct {
	sync.WaitGroup
	s []shutdown
}

// Process - type signature for background process
// and type that implements this Run is a process
type Process interface {
	Run(args interface{}, shutdown <-chan struct{})
}

// Processes - list of processes to start
type Processes []Process

// Start - start up a set of background processes
// all with the same arg value
func Start(processes Processes, args interface{}) *T {

	register := new(T)
	register.WaitGroup = sync.WaitGroup{}
	register.s = make([]shutdown, len(processes))

	// start each background
	for i, p := range processes {
		shutdown := make(chan struct{})
		finished := make(chan struct{})
		register.s[i].shutdown = shutdown
		register.s[i].finished = finished
		register.Add(1)
		go func(p Process, shutdown <-chan struct{}, finished chan<- struct{}) {
			p.Run(args, shutdown)
			register.Done()
			// flag for the stop routine to wait for shutdown
			close(finished)
		}(p, shutdown, finished)
	}
	return register
}

// Stop - stop a set of background processes
func (t *T) Stop() {

	if t == nil {
		return
	}

	// trigger shutdown of all background tasks
	for _, shutdown := range t.s {
		close(shutdown.shutdown)
	}
}

// StopAndWait will notify all processes to shutdown by closing shutdown channel
// and wait until all processes be stopped.
func (t *T) StopAndWait() {
	t.Stop()
	t.Wait()
}
