// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"sync"
	"time"
)

// structure to hold a client connection
//
// prefix:
//   REQ socket this adds an item before send
//   SUB socket this adds/changes subscription
type Client struct {
	sync.Mutex

	publicKey       []byte
	privateKey      []byte
	serverPublicKey []byte
	address         string
	prefix          string
	v6              bool
	socketType      zmq.Type
	socket          *zmq.Socket
	poller          *Poller
	events          zmq.State
	timeout         time.Duration
	timestamp       time.Time
}

const (
	publicKeySize  = 32
	privateKeySize = 32
	identifierSize = 32
)

type globalClientDataType struct {
	sync.Mutex
	clients map[*zmq.Socket]*Client
}

var globalClientData = globalClientDataType{
	clients: make(map[*zmq.Socket]*Client),
}

// create a client socket ususlly of type zmq.REQ or zmq.SUB
func NewClient(socketType zmq.Type, privateKey []byte, publicKey []byte, timeout time.Duration) (*Client, error) {

	if len(publicKey) != publicKeySize {
		return nil, fault.ErrInvalidPublicKey
	}
	if len(privateKey) != privateKeySize {
		return nil, fault.ErrInvalidPrivateKey
	}

	client := &Client{
		publicKey:       make([]byte, publicKeySize),
		privateKey:      make([]byte, privateKeySize),
		serverPublicKey: make([]byte, publicKeySize),
		address:         "",
		v6:              false,
		socketType:      socketType,
		socket:          nil,
		poller:          nil,
		events:          0,
		timeout:         timeout,
		timestamp:       time.Now(),
	}
	copy(client.privateKey, privateKey)
	copy(client.publicKey, publicKey)
	return client, nil
}

// create a socket and connect to specific server with specifed key
func (client *Client) openSocket() error {
	client.Lock()
	defer client.Unlock()

	socket, err := zmq.NewSocket(client.socketType)
	if nil != err {
		return err
	}

	// create a secure random identifier
	randomIdBytes := make([]byte, identifierSize)
	_, err = rand.Read(randomIdBytes)
	if nil != err {
		return err
	}
	randomIdentifier := string(randomIdBytes)

	// set up as client
	err = socket.SetCurveServer(0)
	if nil != err {
		goto failure
	}
	err = socket.SetCurvePublickey(string(client.publicKey))
	if nil != err {
		goto failure
	}
	err = socket.SetCurveSecretkey(string(client.privateKey))
	if nil != err {
		goto failure
	}

	// local identitity is a random value
	err = socket.SetIdentity(randomIdentifier)
	if nil != err {
		goto failure
	}

	// destination identity is its public key
	err = socket.SetCurveServerkey(string(client.serverPublicKey))
	if nil != err {
		goto failure
	}

	// only queue messages sent to connected peers
	socket.SetImmediate(true)

	// zero => do not set timeout
	if 0 != client.timeout {
		err = socket.SetSndtimeo(client.timeout)
		if nil != err {
			goto failure
		}
		err = socket.SetRcvtimeo(client.timeout)
		if nil != err {
			goto failure
		}
	}
	err = socket.SetLinger(100 * time.Millisecond)
	if nil != err {
		goto failure
	}

	// stype specific options
	switch client.socketType {
	case zmq.REQ:
		err = socket.SetReqCorrelate(1)
		if nil != err {
			goto failure
		}
		err = socket.SetReqRelaxed(1)
		if nil != err {
			goto failure
		}

	case zmq.SUB:
		// set subscription prefix - empty => receive everything
		err = socket.SetSubscribe(client.prefix)
		if nil != err {
			goto failure
		}

	default:
	}

	err = socket.SetTcpKeepalive(1)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveCnt(5)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveIdle(60)
	if nil != err {
		goto failure
	}
	err = socket.SetTcpKeepaliveIntvl(60)
	if nil != err {
		goto failure
	}

	// ***** FIX THIS: enabling this causes complete failure
	// ***** FIX THIS: socket disconnects, perhaps after IVL value
	// heartbeat (constants from socket.go)
	// err = socket.SetHeartbeatIvl(heartbeatInterval)
	// if nil != err {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTimeout(heartbeatTimeout)
	// if nil != err {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTtl(heartbeatTTL)
	// if nil != err {
	// 	goto failure
	// }

	// see socket.go for constants
	err = socket.SetMaxmsgsize(maximumPacketSize)
	if nil != err {
		goto failure
	}

	// set IPv6 state before connect
	err = socket.SetIpv6(client.v6)
	if nil != err {
		goto failure
	}

	// new connection
	err = socket.Connect(client.address)
	if nil != err {
		goto failure
	}

	client.socket = socket

	// register client globally
	globalClientData.Lock()
	globalClientData.clients[socket] = client
	globalClientData.Unlock()

	// add to poller
	if nil != client.poller {
		client.poller.Add(client.socket, client.events)
	}
	return nil
failure:
	socket.Close()
	return err
}

// destroy the socket, but leave other connection info so can reconnect
// to the same endpoint again
func (client *Client) closeSocket() error {
	client.Lock()
	defer client.Unlock()

	if nil == client.socket {
		return nil
	}

	// stop any polling
	if nil != client.poller {
		client.poller.Remove(client.socket)
	}

	// if already connected, disconnect first
	if "" != client.address {
		client.socket.Disconnect(client.address)
	}

	// unregister client globally
	globalClientData.Lock()
	delete(globalClientData.clients, client.socket)
	globalClientData.Unlock()

	// close socket
	err := client.socket.Close()
	client.socket = nil
	return err
}

// disconnect old address and connect to new
func (client *Client) Connect(conn *util.Connection, serverPublicKey []byte, prefix string) error {

	// if already connected, disconnect first
	err := client.closeSocket()
	if nil != err {
		return err
	}
	client.address = ""
	client.prefix = prefix

	// small delay to allow any backgroud socket closing
	// and to restrict rate of reconnection
	time.Sleep(5 * time.Millisecond)

	copy(client.serverPublicKey, serverPublicKey)

	client.address, client.v6 = conn.CanonicalIPandPort("tcp://")

	client.timestamp = time.Now()

	return client.openSocket()
}

// check if connected to a node
func (client *Client) IsConnected() bool {
	return "" != client.address
}

// check if connected to a specific node
func (client *Client) IsConnectedTo(serverPublicKey []byte) bool {
	return bytes.Equal(client.serverPublicKey, serverPublicKey)
}

// // check if not connected to any node
// func (client *Client) IsDisconnected() bool {
// 	return "" == client.address
// }

// // get the age of connection
// func (client *Client) Age() time.Duration {
// 	return time.Since(client.timestamp)
// }

// close and reopen the connection
func (client *Client) Reconnect() error {
	_, err := client.ReconnectReturningSocket()
	return err
}

// close and reopen the connection
func (client *Client) ReconnectReturningSocket() (*zmq.Socket, error) {

	err := client.closeSocket()
	if nil != err {
		return nil, err
	}
	err = client.openSocket()
	if nil != err {
		return nil, err
	}
	return client.socket, nil
}

// disconnect old address and close
func (client *Client) Close() error {
	return client.closeSocket()
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
	client.Lock()
	defer client.Unlock()

	if nil == client.socket || "" == client.address {
		return fault.ErrNotConnected
	}

	if 0 == len(items) {
		logger.Panicf("zmqutil.Client.Send no arguments provided")
	}

	if "" != client.prefix {
		_, err := client.socket.Send(client.prefix, zmq.SNDMORE)
		if nil != err {
			return err
		}
	}

	n := len(items) - 1
	a := items[:n]
	final := items[n] // just the final item

	for i, item := range a {
		switch it := item.(type) {
		case string:
			_, err := client.socket.Send(it, zmq.SNDMORE)
			if nil != err {
				return err
			}
		case []byte:
			_, err := client.socket.SendBytes(it, zmq.SNDMORE)
			if nil != err {
				return err
			}
		case [][]byte:
			for _, sub := range it {
				_, err := client.socket.SendBytes(sub, zmq.SNDMORE)
				if nil != err {
					return err
				}
			}
		default:
			logger.Panicf("zmqutil.Client.Send cannot send[%d]: %#v", i, item)
		}
	}

	switch it := final.(type) {
	case string:
		_, err := client.socket.Send(it, 0)
		if nil != err {
			return err
		}
	case []byte:
		_, err := client.socket.SendBytes(it, 0)
		if nil != err {
			return err
		}
	case [][]byte:
		if 0 == len(it) {
			logger.Panicf("zmqutil.Client.Send empty [][]byte")
		}
		n := len(it) - 1
		a := it[:n]
		last := it[n] // just the final item []byte

		for _, sub := range a {
			_, err := client.socket.SendBytes(sub, zmq.SNDMORE)
			if nil != err {
				return err
			}
		}
		_, err := client.socket.SendBytes(last, 0)
		if nil != err {
			return err
		}

	default:
		logger.Panicf("zmqutil.Client.Send cannot send[%d]: %#v", n, final)
	}

	return nil
}

// receive a reply
func (client *Client) Receive(flags zmq.Flag) ([][]byte, error) {
	client.Lock()
	defer client.Unlock()

	if nil == client.socket || "" == client.address {
		return nil, fault.ErrNotConnected
	}
	data, err := client.socket.RecvMessageBytes(flags)
	return data, err
}

// add poller to client
func (client *Client) BeginPolling(poller *Poller, events zmq.State) *zmq.Socket {

	// if poller changed
	if nil != client.poller && nil != client.socket {
		client.poller.Remove(client.socket)
	}

	// add to new poller
	client.poller = poller
	client.events = events
	if nil != client.socket {
		poller.Add(client.socket, events)
	}
	return client.socket
}

// to string
func (client *Client) String() string {
	return client.address
}

type Connected struct {
	Address string `json:"address"`
	Server  string `json:"server"`
}

// to string
func (client *Client) ConnectedTo() *Connected {

	if "" == client.address {
		return nil
	}
	return &Connected{
		Address: client.address,
		Server:  hex.EncodeToString(client.serverPublicKey),
	}
}

// find the client corresponding to a socket
func ClientFromSocket(socket *zmq.Socket) *Client {
	globalClientData.Lock()
	client := globalClientData.clients[socket]
	globalClientData.Unlock()
	return client
}

// Return a basic information string for debug
func (client *Client) BasicInfo() string {
	s := fmt.Sprintf("serverPublicKey:%x address:%s publicKey:%x  prefix:%s v6:%v socketType:%d ts:%v timeout duration:%s",
		client.serverPublicKey, client.address, client.publicKey, client.prefix, client.v6, client.socketType, client.timestamp, client.timeout.String())

	return s
}

// Return ServerPublicKey
func (client *Client) GetServerPublicKey() []byte {
	return client.serverPublicKey
}

// Clear Server fields for reusing the client
func (client *Client) ResetServer() error {
	err := client.closeSocket()
	if nil != err {
		return err
	}
	client.serverPublicKey = make([]byte, publicKeySize)
	client.address = ""
	client.v6 = false
	return nil
}
