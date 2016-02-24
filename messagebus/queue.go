// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messagebus

// internal constants
const (
	queueSize = 1000
)

type Message struct {
	From string
	Item interface{}
}

var (
	// for queueing data
	queue = make(chan Message, queueSize)
)

// data to queue
func Send(from string, item interface{}) {
	queue <- Message{
		From: from,
		Item: item,
	}
}

// channel to read from
func Chan() <-chan Message {
	return queue
}
