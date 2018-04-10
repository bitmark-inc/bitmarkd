// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	zmq "github.com/pebbe/zmq4"
)

// return a socket connection to the monitoring channel of another socket
// for connection state signalling
// a unique inproc://name must be provided for each use
func NewMonitor(socket *zmq.Socket, connection string, event zmq.Event) (*zmq.Socket, error) {

	err := socket.Monitor(connection, event)
	if err != nil {
		return nil, err
	}

	mon, err := zmq.NewSocket(zmq.PAIR)
	if err != nil {
		return nil, err
	}

	err = mon.Connect(connection)
	if err != nil {
		mon.Close()
		return nil, err
	}

	return mon, nil
}
