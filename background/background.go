// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package background

// the shudown and completed type for a background
type shutdown struct {
	shutdown chan bool
	finished chan bool
}

// handle type
type T struct {
	s []shutdown
}

// type signature for background process
type Process func(args interface{}, shutdown <-chan bool, done chan<- bool)

// list of processes to start
type Processes []Process

// start up a set of background processes
func Start(processes Processes, args interface{}) *T {

	register := new(T)
	register.s = make([]shutdown, len(processes))

	// start each background
	for i, p := range processes {
		shutdown := make(chan bool)
		finished := make(chan bool)
		register.s[i].shutdown = shutdown
		register.s[i].finished = finished
		go p(args, shutdown, finished)
	}
	return register
}

// stop a set of background processes
func Stop(t *T) {

	// shutdown all background tasks
	for _, shutdown := range t.s {
		close(shutdown.shutdown)
	}

	// wait for finished
	for _, shutdown := range t.s {
		<-shutdown.finished
	}
}
