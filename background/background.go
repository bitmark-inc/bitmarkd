// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package background

// the shutdown and completed type for a background
type shutdown struct {
	shutdown chan struct{}
	finished chan struct{}
}

// T - handle type for the stop
type T struct {
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
	register.s = make([]shutdown, len(processes))

	// start each background
	for i, p := range processes {
		shutdown := make(chan struct{})
		finished := make(chan struct{})
		register.s[i].shutdown = shutdown
		register.s[i].finished = finished
		go func(p Process, shutdown <-chan struct{}, finished chan<- struct{}) {
			// pass the shutdown to the Run loop for shutdown signalling
			p.Run(args, shutdown)
			// flag for the stop routine to wait for shutdown
			close(finished)
		}(p, shutdown, finished)
	}
	return register
}

// Stop - stop a set of background processes
func (t *T) Stop() {

	if nil == t {
		return
	}

	// trigger shutdown of all background tasks
	for _, shutdown := range t.s {
		close(shutdown.shutdown)
	}
}
