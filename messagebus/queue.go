// Copyright (c) 2014-2019 Bitmark Inc.
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

// Message - message to put into a queue
type Message struct {
	Command    string   // type of packed data
	Parameters [][]byte // array of parameters
}

// Queue - a 1:1 queue
type Queue struct {
	c    chan Message
	size int
	used bool
}

// BroadcastQueue - a 1:M queue
// out is synchronous, so messages to routines not waiting are dropped
type BroadcastQueue struct {
	in          chan Message
	out         []chan Message
	defaultSize int
}

// the exported message queues and their sizes
// any item with a size option will be allocated that size
// absent then default size is used
type busses struct {
	Broadcast  *BroadcastQueue `size:"1000"` // to broadcast to other nodes
	Connector  *Queue          `size:"50"`   // to control connector
	Announce   *Queue          `size:"50"`   // to control the announcer
	Blockstore *Queue          `size:"50"`   // to sequentially store blocks
	TestQueue  *Queue          `size:"50"`   // for testing use
}

// Bus - all available message queues
var Bus busses

// cache for the delivered message
var deliveredMessage map[string]Message

// initialise all queues with preset size
func init() {

	// this will be a struct type
	busType := reflect.TypeOf(Bus)

	// get write access by using pointer + Elem()
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

		switch qt := busValue.Field(i).Type(); qt {

		case reflect.TypeOf((*BroadcastQueue)(nil)):
			q := &BroadcastQueue{
				in:          make(chan Message, queueSize),
				out:         make([]chan Message, 0, 10),
				defaultSize: queueSize,
			}
			go q.multicast()

			newQueue := reflect.ValueOf(q)
			busValue.Field(i).Set(newQueue)

		case reflect.TypeOf((*Queue)(nil)):
			q := &Queue{
				c:    make(chan Message, queueSize),
				size: queueSize,
				used: false,
			}
			newQueue := reflect.ValueOf(q)
			busValue.Field(i).Set(newQueue)
		default:
			panic(fmt.Sprintf("queue type: %q is not handled", qt))
		}
	}
}

// Send - send a message to a 1:1 queue
// but only if listener is connected
func (queue *Queue) Send(command string, parameters ...[]byte) {
	queue.c <- Message{
		Command:    command,
		Parameters: parameters,
	}
}

// Chan - channel to read from 1:1 queue
// can only be called once
func (queue *Queue) Chan() <-chan Message {
	if queue.used {
		panic("cannot get a second receive channel from a 1:1 queue")
	}
	queue.used = true
	return queue.c
}

// Release - give the channel back
func (queue *Queue) Release() {
	queue.used = false
	close(queue.c)
	queue.c = make(chan Message, queue.size)
}

// Send - send a message to a 1:M queue
func (queue *BroadcastQueue) Send(command string, parameters ...[]byte) {
	m := Message{
		Command:    command,
		Parameters: parameters,
	}

	if isCached(m) {
		// Drop the message that already cached
		return
	}

	queue.in <- m

	// Ignore caching rpc/peer command
	if "rpc" == command || "peer" == command {
		return
	}
	cache(m)
}

// Chan - get a new channel to read from a 1:M queue
// each call gets a distinct channel
func (queue *BroadcastQueue) Chan(size int) <-chan Message {
	if size < 0 {
		size = queue.defaultSize
	}
	c := make(chan Message, size)
	queue.out = append(queue.out, c)
	return c
}

// DropCache - drop the item from cache
func DropCache(m Message) {
	if !isCached(m) {
		return
	}

	delete(deliveredMessage, m.packHex())
}

// cache the delivered message
func cache(m Message) {

	if isCached(m) {
		return
	}

	// store the message to cache
	if nil == deliveredMessage {
		deliveredMessage = make(map[string]Message)
	}
	deliveredMessage[m.packHex()] = m
}

// pack Message
func (m Message) pack() []byte {
	s := make([]byte, 0)
	s = append(s, []byte(m.Command)...)
	for _, a := range m.Parameters {
		s = append(s, a...)
	}
	return s
}

// pack Message in hex string
func (m Message) packHex() string {
	return fmt.Sprintf("%x", m.pack())
}

// check whether Message is cached
func isCached(m Message) (isCached bool) {
	if _, ok := deliveredMessage[m.packHex()]; ok {
		isCached = true
	}
	return
}

// background processing for the 1:M queue
//
// if an outgoing queue is full just drop the message
// to avoid blocking
func (queue *BroadcastQueue) multicast() {
	c := queue.in
	for {
		data := <-c
		for _, out := range queue.out {
			select {
			case out <- data:
			default:
			}
		}

	}
}
