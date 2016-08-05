// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	zmq "github.com/pebbe/zmq4"
	"time"
)

// structure to hold a client connection
type Client struct {
	publicKey []byte
	address   string
	socket    *zmq.Socket
	timestamp time.Time
}

// create a cliet socket ususlly of type zmq.REQ or zmq.SUB
func NewClient(socketType zmq.Type, privateKey []byte, publicKey []byte, timeout time.Duration) (*Client, error) {
	socket, err := zmq.NewSocket(socketType)
	if nil != err {
		return nil, err
	}

	// set up as client
	socket.SetCurveServer(0)
	socket.SetCurvePublickey(string(publicKey))
	socket.SetCurveSecretkey(string(privateKey))

	socket.SetIdentity(string(publicKey)) // just use public key for identity

	// // basic socket options
	// socket.SetIpv6(true) // do not set here defer to connect
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer

	// zero => do not set timeout
	if 0 != timeout {
		socket.SetSndtimeo(timeout)
		socket.SetRcvtimeo(timeout)
	}
	socket.SetLinger(0)

	// stype specific options
	switch socketType {
	case zmq.REQ:
		socket.SetReqCorrelate(1)
		socket.SetReqRelaxed(1)

	case zmq.SUB:
		// set subscription prefix - empty => receive everything
		socket.SetSubscribe("")

	default:
	}

	client := &Client{
		publicKey: make([]byte, 32),
		address:   "",
		socket:    socket,
		timestamp: time.Now(),
	}
	return client, nil
}

// disconnect old address and connect to new
func (client *Client) Connect(conn *util.Connection, serverPublicKey []byte) error {

	// if already connected, disconnect first
	if "" != client.address {
		err := client.socket.Disconnect(client.address)
		if nil != err {
			return err
		}
	}
	client.address = ""

	err := client.socket.SetCurveServerkey(string(serverPublicKey))
	if nil != err {
		return err
	}

	connectTo, v6 := conn.CanonicalIPandPort("tcp://")

	// set IPv6 state before connect
	err = client.socket.SetIpv6(v6)
	if nil != err {
		return err
	}

	// new connection
	err = client.socket.Connect(connectTo)
	if nil != err {
		return err
	}

	// record details
	client.address = connectTo
	copy(client.publicKey, serverPublicKey)
	client.timestamp = time.Now()

	return nil
}

// check if connected to a node
func (client *Client) IsConnected() bool {
	return "" != client.address
}

// check if connected to a specific node
func (client *Client) IsConnectedTo(serverPublicKey []byte) bool {
	return bytes.Equal(client.publicKey, serverPublicKey)
}

// check if not connected to any node
func (client *Client) IsDisconnected() bool {
	return "" == client.address
}

// get the age of connection
func (client *Client) Age() time.Duration {
	return time.Since(client.timestamp)
}

// close and reopen the connection
func (client *Client) Reconnect() error {
	if "" == client.address {
		return nil
	}
	err := client.socket.Disconnect(client.address)
	if nil != err {
		return err
	}
	err = client.socket.Connect(client.address)
	if nil != err {
		return err
	}
	return nil
}

// disconnect old address and close
func (client *Client) Close() error {
	// if already connected, disconnect first
	if "" != client.address {
		client.socket.Disconnect(client.address)
	}
	client.address = ""

	// close socket
	err := client.socket.Close()
	client.socket = nil

	return err
}

// disconnect old addresses and close all
func CloseClients(clients []*Client) {
	for _, client := range clients {
		if nil != client {
			client.Close()
		}
	}
}

// send a message
func (client *Client) Send(items ...interface{}) error {
	if "" == client.address {
		return fault.ErrNotConnected
	}

	last := len(items) - 1
	for i, item := range items {

		flag := zmq.SNDMORE
		if i == last {
			flag = 0
		}
		switch item.(type) {
		case string:
			_, err := client.socket.Send(item.(string), flag)
			if nil != err {
				return err
			}
		case []byte:
			_, err := client.socket.SendBytes(item.([]byte), flag)
			if nil != err {
				return err
			}
		}
	}
	return nil
}

// receive a reply
func (client *Client) Receive(flags zmq.Flag) ([][]byte, error) {
	if "" == client.address {
		return nil, fault.ErrNotConnected
	}
	data, err := client.socket.RecvMessageBytes(flags)
	return data, err
}

// add to a poller
func (client *Client) Add(poller *zmq.Poller, state zmq.State) int {
	return poller.Add(client.socket, state)
}
