// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

type ClientIntf interface {
	Connect(conn *util.Connection, serverPublicKey []byte, prefix string) error
	IsConnected() bool
	IsConnectedTo(serverPublicKey []byte) bool
	Reconnect() error
	ReconnectReturningSocket() (*zmq.Socket, error)
	ReconnectOpenedSocket() error
	GetSocket() *zmq.Socket
	Close() error
	Send(items ...interface{}) error
	Receive(flags zmq.Flag) ([][]byte, error)
	BeginPolling(poller *Poller, events zmq.State) *zmq.Socket
	String() string
	ConnectedTo() *Connected
	GoString() string
	GetMonitorSocket() *zmq.Socket
	ServerPublicKey() []byte
	ResetServer() error
}

// Client - structure to hold a client connection
//
// prefix:
//   REQ socket this adds an item before send
//   SUB socket this adds/changes subscription
type Client struct {
	ClientIntf

	sync.Mutex

	publicKey       []byte
	privateKey      []byte
	serverPublicKey []byte
	address         string
	prefix          string
	v6              bool
	socketType      zmq.Type
	socket          *zmq.Socket
	timeout         time.Duration
	timestamp       time.Time
	number          uint64
	monitorEvents   zmq.Event
	monitorShutdown chan struct{}
	monitorStopped  chan struct{}
	queue           chan Event
}

type Event struct {
	Event   zmq.Event
	Address string
	Value   int
}

const (
	publicKeySize  = 32
	privateKeySize = 32
	identifierSize = 32
	tcpPrefix      = "tcp://"
	monitorFormat  = "inproc://client%d-%d-monitor"
	signalFormat   = "inproc://client%d-%d-signal"
)

// atomically incremented counter for monitor names
var clientCounter counter.Counter

// atomically incremented counter for monitor revisions
// to allow ZeroMQ to finish closing the old name when generating a new one
var sequenceCounter counter.Counter

// NewClient - create a client socket ususlly of type zmq.REQ or zmq.SUB
func NewClient(
	socketType zmq.Type,
	privateKey []byte,
	publicKey []byte,
	timeout time.Duration,
	events zmq.Event,
) (*Client, <-chan Event, error) {

	if len(publicKey) != publicKeySize {
		return nil, nil, fault.ErrInvalidPublicKey
	}
	if len(privateKey) != privateKeySize {
		return nil, nil, fault.ErrInvalidPrivateKey
	}

	n := clientCounter.Increment()

	queue := make(chan Event, 1)

	client := &Client{
		publicKey:       make([]byte, publicKeySize),
		privateKey:      make([]byte, privateKeySize),
		serverPublicKey: make([]byte, publicKeySize),
		address:         "",
		v6:              false,
		socketType:      socketType,
		socket:          nil,
		timeout:         timeout,
		timestamp:       time.Now(),
		number:          n,
		monitorEvents:   events,
		monitorShutdown: nil,
		monitorStopped:  nil,
		queue:           queue,
	}
	copy(client.privateKey, privateKey)
	copy(client.publicKey, publicKey)
	return client, queue, nil
}

// create a socket and connect to specific server with public key
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

	if 0 != client.monitorEvents {
		client.monitorShutdown = make(chan struct{})
		client.monitorStopped = make(chan struct{})
		go client.poller(client.monitorShutdown, client.monitorStopped)
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

	// if already connected, disconnect first
	if "" != client.address {
		client.socket.Disconnect(client.address)
	}

	// close sockets
	err := client.socket.Close()
	client.socket = nil
	if nil != client.monitorShutdown {
		close(client.monitorShutdown)
		client.monitorShutdown = nil
		<-client.monitorStopped
		client.monitorStopped = nil
		// small delay to allow any background socket closing
		// and to restrict rate of reconnection
		time.Sleep(5 * time.Millisecond)
	}

	return err
}

// Connect - disconnect old address and connect to new
func (client *Client) Connect(conn *util.Connection, serverPublicKey []byte, prefix string) error {

	// if already connected, disconnect first
	err := client.closeSocket()
	if nil != err {
		return err
	}
	client.address = ""
	client.prefix = prefix

	// small delay to allow any background socket closing
	// and to restrict rate of reconnection
	time.Sleep(5 * time.Millisecond)

	copy(client.serverPublicKey, serverPublicKey)

	client.address, client.v6 = conn.CanonicalIPandPort(tcpPrefix)

	client.timestamp = time.Now()

	return client.openSocket()
}

// IsConnected - check if connected to a node
func (client *Client) IsConnected() bool {
	return "" != client.address && nil != client.socket
}

// IsConnectedTo - check if connected to a specific node
func (client *Client) IsConnectedTo(serverPublicKey []byte) bool {
	return bytes.Equal(client.serverPublicKey, serverPublicKey)
}

// Reconnect - close and reopen the connection
func (client *Client) Reconnect() error {

	err := client.closeSocket()
	if nil != err {
		return err
	}
	err = client.openSocket()
	if nil != err {
		return err
	}
	return nil
}

// Close - disconnect old address and close
func (client *Client) Close() error {
	err := client.closeSocket()
	client.serverPublicKey = make([]byte, publicKeySize)
	client.address = ""
	client.v6 = false
	return err
}

// CloseClients - disconnect old addresses and close all
func CloseClients(clients []*Client) {
	for _, client := range clients {
		if nil != client {
			client.Close()
		}
	}
}

// Send - send a message
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

// Receive - receive a reply
func (client *Client) Receive(flags zmq.Flag) ([][]byte, error) {
	client.Lock()
	defer client.Unlock()

	if nil == client.socket || "" == client.address {
		return nil, fault.ErrNotConnected
	}
	data, err := client.socket.RecvMessageBytes(flags)
	return data, err
}

// Connected - representation of a connected server
type Connected struct {
	Address string `json:"address"`
	Server  string `json:"server"`
}

// ConnectedTo - return representation of client connection
func (client *Client) ConnectedTo() *Connected {

	if "" == client.address {
		return nil
	}
	return &Connected{
		Address: client.address,
		Server:  hex.EncodeToString(client.serverPublicKey),
	}
}

// String - return a string description of a client
func (client *Client) String() string {
	return client.address
}

// GoString - return a basic information string for debugging purposes
func (client *Client) GoString() string {
	s := fmt.Sprintf(
		"server public key: %x  address: %s  public key: %x  prefix: %s  v6: %t  socket type: %d  ts: %v  timeout duration: %s",
		client.serverPublicKey,
		client.address,
		client.publicKey,
		client.prefix,
		client.v6,
		client.socketType,
		client.timestamp,
		client.timeout.String())

	return s
}

// ServerPublicKey - return server's public key
func (client *Client) ServerPublicKey() []byte {
	return client.serverPublicKey
}

func (client *Client) poller(shutdown <-chan struct{}, stopped chan<- struct{}) {

	n := sequenceCounter.Increment()
	monitorConnection := fmt.Sprintf(monitorFormat, client.number, n)
	monitorSignal := fmt.Sprintf(signalFormat, client.number, n)

	sigReceive, sigSend, err := NewSignalPair(monitorSignal)
	if nil != err {
		logger.Panicf("cannot create signal for: %s  error: %s", monitorSignal, err)
	}

	m, err := NewMonitor(client.socket, monitorConnection, client.monitorEvents)
	if nil != err {
		logger.Panicf("cannot create monitor for: %s  error: %s", monitorConnection, err)
	}

	go func(m *zmq.Socket, queue chan<- Event) {
		poller := NewPoller()

		poller.Add(m, zmq.POLLIN)

		poller.Add(sigReceive, zmq.POLLIN)

	loop:
		//log.Debug("start polling…")
		for {
			//log.Debug("waiting…")

			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case sigReceive:
					break loop
				default:
					handleEvent(s, queue)
				}
			}
		}

		poller.Remove(sigReceive)
		poller.Remove(m)
		sigReceive.Close()
		m.Close()
		close(stopped)
		//log.Debug("stopped polling")
	}(m, client.queue)

	// wait here for signal
	<-shutdown

	sigSend.SendMessage("stop")
	sigSend.Close()
}

// process the socket events
func handleEvent(s *zmq.Socket, queue chan<- Event) error {
	ev, addr, v, err := s.RecvEvent(0)
	if nil != err {
		return err
	}

	e := Event{
		Event:   ev,
		Address: addr,
		Value:   v,
	}

	select {
	case queue <- e:
	default:
	}

	return nil
}
