// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messagebus

import (
	"fmt"
	"reflect"
	"strconv"
)

// internal constants
const (
	defaultQueueSize = 1000 // if specific size is not specified
)

// message to put into a queue
type Message struct {
	Command    string   // type of packed data
	Parameters [][]byte // array of parameters
}

// structure of an individual queue
type Queue struct {
	c chan Message
}

// the exported message queues and their sizes
// any item with a size option will be allocated that size
// absent then default size is used
type busses struct {
	Broadcast  *Queue `size:"1000"` // to broadcast to other nodes
	Subscriber *Queue `size:"50"`   // to control subscriber
	Connector  *Queue `size:"50"`   // to control connector
	Blockstore *Queue `size:"50"`   // to sequentially store blocks
}

// the instance
var Bus busses

// initialise all queues with preset size
func init() {

	// this will be a struct type
	busType := reflect.TypeOf(Bus)

	// get write acces by using pointer + Elem()
	busValue := reflect.ValueOf(&Bus).Elem()

	// scan each field
	for i := 0; i < busType.NumField(); i += 1 {

		fieldInfo := busType.Field(i)

		sizeTag := fieldInfo.Tag.Get("size")

		queueSize := defaultQueueSize

		// if size specified and valid positive integer override default
		if len(sizeTag) > 0 {
			s, err := strconv.Atoi(sizeTag)
			if nil == err && s > 0 {
				queueSize = s
			} else {
				m := fmt.Sprintf("queue: %v  has invalid size: %q", fieldInfo, sizeTag)
				panic(m)
			}
		}
		q := &Queue{
			c: make(chan Message, queueSize),
		}
		newQueue := reflect.ValueOf(q)

		busValue.Field(i).Set(newQueue)
	}
}

// send a message to a queue
func (queue *Queue) Send(command string, parameters ...[]byte) {
	queue.c <- Message{
		Command:    command,
		Parameters: parameters,
	}
}

// channel to read from
func (queue *Queue) Chan() <-chan Message {
	return queue.c
}
