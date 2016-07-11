// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messagebus

// internal constants
const (
	queueSize = 1000
)

type Message struct {
	Kind string // type of packed data
	Data []byte // data bytes
}

var (
	// for queueing data
	queue = make(chan Message, queueSize)
)

// data to queue
func Send(kind string, data []byte) {
	queue <- Message{
		Kind: kind,
		Data: data,
	}
}

// channel to read from
func Chan() <-chan Message {
	return queue
}
