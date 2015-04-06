// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messagebus

// internal constants
const (
	queueSize = 100
)

var (
	// for queueing data
	queue = make(chan interface{}, queueSize)
)

// data to queue
func Send(item interface{}) {
	queue <- item
}

// channel to read from
func Chan() <-chan interface{} {
	return queue
}
