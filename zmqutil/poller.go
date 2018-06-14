// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"
)

// structure to hold a poller
type Poller struct {
	sync.Mutex
	sockets map[*zmq.Socket]zmq.State
	poller  *zmq.Poller
}

// create a poller
// this is just to encapsulate the zmq poller to allow removal of a socket from a socket
func NewPoller() *Poller {
	return &Poller{
		sockets: make(map[*zmq.Socket]zmq.State),
		poller:  zmq.NewPoller(),
	}
}

// add a socket to a poller
func (poller *Poller) Add(socket *zmq.Socket, events zmq.State) {

	poller.Lock()
	defer poller.Unlock()

	// protect against duplicate add
	if _, ok := poller.sockets[socket]; ok {
		return
	}

	// preserve the event mask
	poller.sockets[socket] = events

	// add to the internal poller
	poller.poller.Add(socket, events)
}

// remove a socket from a poller
func (poller *Poller) Remove(socket *zmq.Socket) {

	poller.Lock()
	defer poller.Unlock()

	// protect against duplicate remove
	if _, ok := poller.sockets[socket]; !ok {
		return
	}

	// remove the socket
	delete(poller.sockets, socket)
	poller.poller.RemoveBySocket(socket)
}

// perform a poll
func (poller *Poller) Poll(timeout time.Duration) ([]zmq.Polled, error) {
	poller.Lock()
	p := poller.poller
	poller.Unlock()
	polled, err := p.Poll(timeout)
	return polled, err
}
