// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

// re-export zmq constants
const (
	EVENT_CONNECTED       = zmq.EVENT_CONNECTED
	EVENT_CONNECT_DELAYED = zmq.EVENT_CONNECT_DELAYED
	EVENT_CONNECT_RETRIED = zmq.EVENT_CONNECT_RETRIED
	EVENT_LISTENING       = zmq.EVENT_LISTENING
	EVENT_BIND_FAILED     = zmq.EVENT_BIND_FAILED
	EVENT_ACCEPTED        = zmq.EVENT_ACCEPTED
	EVENT_ACCEPT_FAILED   = zmq.EVENT_ACCEPT_FAILED
	EVENT_CLOSED          = zmq.EVENT_CLOSED
	EVENT_CLOSE_FAILED    = zmq.EVENT_CLOSE_FAILED
	EVENT_DISCONNECTED    = zmq.EVENT_DISCONNECTED
	EVENT_MONITOR_STOPPED = zmq.EVENT_MONITOR_STOPPED
	EVENT_ALL             = zmq.EVENT_ALL
	// ***** FIX THIS: not defined by zmq
	EVENT_HANDSHAKE_FAILED_NO_DETAIL = 0x0800
	EVENT_HANDSHAKE_SUCCEEDED        = 0x1000
	EVENT_HANDSHAKE_FAILED_PROTOCOL  = 0x2000
	EVENT_HANDSHAKE_FAILED_AUTH      = 0x4000
)

// Client - structure to hold a client connection
type Client interface {
	Close() error
	Connect(conn *util.Connection, serverPublicKey []byte, prefix string) error
	ConnectedTo() *Connected
	GoString() string
	IsConnected() bool
	IsConnectedTo(serverPublicKey []byte) bool
	Reconnect() error
	Receive(flags zmq.Flag) ([][]byte, error)
	Send(items ...interface{}) error
	ServerPublicKey() []byte
	String() string
}

// clientData - structure to hold a client connection
//
// prefix:
//
//	REQ socket this adds an item before send
//	SUB socket this adds/changes subscription
type clientData struct {
	Client

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
	queue           chan Event
	monitorEvents   zmq.Event
	monitorShutdown *zmq.Socket
	// monitorShutdown chan struct{}
	// monitorStopped  chan struct{}
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
) (Client, <-chan Event, error) {

	if len(publicKey) != publicKeySize {
		return nil, nil, fault.InvalidPublicKey
	}
	if len(privateKey) != privateKeySize {
		return nil, nil, fault.InvalidPrivateKey
	}

	n := clientCounter.Increment()

	queue := make(chan Event, 10)

	client := &clientData{
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
		queue:           queue,
		monitorEvents:   events,
		monitorShutdown: nil,
		//monitorStopped:  nil,
	}
	copy(client.privateKey, privateKey)
	copy(client.publicKey, publicKey)
	return client, queue, nil
}

// create a socket and connect to specific server with public key
func (client *clientData) openSocket() error {
	client.Lock()
	defer client.Unlock()

	if client.socket != nil {
		logger.Panicf("socket is not closed")
	}

	// create a secure random identifier
	randomIdBytes := make([]byte, identifierSize)
	_, err := rand.Read(randomIdBytes)
	if err != nil {
		return err
	}
	randomIdentifier := string(randomIdBytes)

	// create a new socket
	socket, err := zmq.NewSocket(client.socketType)
	if err != nil {
		return err
	}

	// all errors after here must goto failure to ensure proper
	// cleanup

	// set up as client
	err = socket.SetCurveServer(0)
	if err != nil {
		goto failure
	}
	err = socket.SetCurvePublickey(string(client.publicKey))
	if err != nil {
		goto failure
	}
	err = socket.SetCurveSecretkey(string(client.privateKey))
	if err != nil {
		goto failure
	}

	// local identitity is a random value
	err = socket.SetIdentity(randomIdentifier)
	if err != nil {
		goto failure
	}

	// destination identity is its public key
	err = socket.SetCurveServerkey(string(client.serverPublicKey))
	if err != nil {
		goto failure
	}

	// only queue messages sent to connected peers
	socket.SetImmediate(true)

	// zero => do not set timeout
	if client.timeout != 0 {
		err = socket.SetSndtimeo(client.timeout)
		if err != nil {
			goto failure
		}
		err = socket.SetRcvtimeo(client.timeout)
		if err != nil {
			goto failure
		}
	}
	err = socket.SetLinger(100 * time.Millisecond)
	if err != nil {
		goto failure
	}

	// stype specific options
	switch client.socketType {
	case zmq.REQ:
		err = socket.SetReqCorrelate(1)
		if err != nil {
			goto failure
		}
		err = socket.SetReqRelaxed(1)
		if err != nil {
			goto failure
		}

	case zmq.SUB:
		// set subscription prefix - empty => receive everything
		err = socket.SetSubscribe(client.prefix)
		if err != nil {
			goto failure
		}

	default:
	}

	err = socket.SetTcpKeepalive(1)
	if err != nil {
		goto failure
	}
	err = socket.SetTcpKeepaliveCnt(5)
	if err != nil {
		goto failure
	}
	err = socket.SetTcpKeepaliveIdle(60)
	if err != nil {
		goto failure
	}
	err = socket.SetTcpKeepaliveIntvl(60)
	if err != nil {
		goto failure
	}

	// set approx reconnect tries to 2 minutes
	// and disable exponential back-off
	// should give around 20 retries in the announce timeout
	err = socket.SetReconnectIvl(2 * time.Minute)
	if err != nil {
		goto failure
	}
	err = socket.SetReconnectIvlMax(0)
	if err != nil {
		goto failure
	}

	// ***** FIX THIS: enabling this causes complete failure
	// ***** FIX THIS: socket disconnects, perhaps after IVL value
	// heartbeat (constants from socket.go)
	// err = socket.SetHeartbeatIvl(heartbeatInterval)
	// if err != nil {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTimeout(heartbeatTimeout)
	// if err != nil {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTtl(heartbeatTTL)
	// if err != nil {
	// 	goto failure
	// }

	// see socket.go for constants
	err = socket.SetMaxmsgsize(maximumPacketSize)
	if err != nil {
		goto failure
	}

	// set IPv6 state before connect
	err = socket.SetIpv6(client.v6)
	if err != nil {
		goto failure
	}

	// new connection
	err = socket.Connect(client.address)
	if err != nil {
		goto failure
	}

	client.socket = socket

	if client.monitorEvents != 0 {

		n := sequenceCounter.Increment()
		monitorConnection := fmt.Sprintf(monitorFormat, client.number, n)
		monitorSignal := fmt.Sprintf(signalFormat, client.number, n)

		sigReceive, sigSend, err := NewSignalPair(monitorSignal)
		if err != nil {
			logger.Panicf("cannot create signal for: %s  error: %s", monitorSignal, err)
		}

		m, err := NewMonitor(client.socket, monitorConnection, client.monitorEvents)
		if err != nil {
			logger.Panicf("cannot create monitor for: %s  error: %s", monitorConnection, err)
		}
		client.monitorShutdown = sigSend

		go poller(m, sigReceive, client.queue)
	}

	return nil
failure:
	socket.Close()
	return err
}

// destroy the socket, but leave other connection info so can reconnect
// to the same endpoint again
func (client *clientData) closeSocket() error {
	client.Lock()
	defer client.Unlock()

	if client.socket == nil {
		return nil
	}

	// if already connected, disconnect first
	if client.address != "" {
		client.socket.Disconnect(client.address)
	}

	// close sockets
	err := client.socket.Close()
	client.socket = nil
	if client.monitorShutdown != nil {
		client.monitorShutdown.SendMessage("stop")
		client.monitorShutdown.Close()
		client.monitorShutdown = nil
		// small delay to allow any background socket closing
		// and to restrict rate of reconnection
		time.Sleep(5 * time.Millisecond)
	}

	return err
}

// Connect - disconnect old address and connect to new
func (client *clientData) Connect(conn *util.Connection, serverPublicKey []byte, prefix string) error {

	// if already connected, disconnect first
	err := client.closeSocket()
	if err != nil {
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
func (client *clientData) IsConnected() bool {
	return client.address != "" && client.socket != nil
}

// IsConnectedTo - check if connected to a specific node
func (client *clientData) IsConnectedTo(serverPublicKey []byte) bool {
	return bytes.Equal(client.serverPublicKey, serverPublicKey)
}

// Reconnect - close and reopen the connection
func (client *clientData) Reconnect() error {

	err := client.closeSocket()
	if err != nil {
		return err
	}
	err = client.openSocket()
	if err != nil {
		return err
	}
	return nil
}

// Close - disconnect old address and close
func (client *clientData) Close() error {
	err := client.closeSocket()
	client.serverPublicKey = make([]byte, publicKeySize)
	client.address = ""
	client.v6 = false
	return err
}

// CloseClients - disconnect old addresses and close all
func CloseClients(clients []Client) {
	for _, client := range clients {
		if client != nil {
			client.Close()
		}
	}
}

// Send - send a message
func (client *clientData) Send(items ...interface{}) error {
	client.Lock()
	defer client.Unlock()

	if client.socket == nil || client.address == "" {
		return fault.NotConnected
	}

	if len(items) == 0 {
		logger.Panicf("zmqutil.Client.Send no arguments provided")
	}

	if client.prefix != "" {
		_, err := client.socket.Send(client.prefix, zmq.SNDMORE)
		if err != nil {
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
			if err != nil {
				return err
			}
		case []byte:
			_, err := client.socket.SendBytes(it, zmq.SNDMORE)
			if err != nil {
				return err
			}
		case [][]byte:
			for _, sub := range it {
				_, err := client.socket.SendBytes(sub, zmq.SNDMORE)
				if err != nil {
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
		if err != nil {
			return err
		}
	case []byte:
		_, err := client.socket.SendBytes(it, 0)
		if err != nil {
			return err
		}
	case [][]byte:
		if len(it) == 0 {
			logger.Panicf("zmqutil.Client.Send empty [][]byte")
		}
		n := len(it) - 1
		a := it[:n]
		last := it[n] // just the final item []byte

		for _, sub := range a {
			_, err := client.socket.SendBytes(sub, zmq.SNDMORE)
			if err != nil {
				return err
			}
		}
		_, err := client.socket.SendBytes(last, 0)
		if err != nil {
			return err
		}

	default:
		logger.Panicf("zmqutil.Client.Send cannot send[%d]: %#v", n, final)
	}

	return nil
}

// Receive - receive a reply
func (client *clientData) Receive(flags zmq.Flag) ([][]byte, error) {
	client.Lock()
	defer client.Unlock()

	if client.socket == nil || client.address == "" {
		return nil, fault.NotConnected
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
func (client *clientData) ConnectedTo() *Connected {

	if client.address == "" {
		return nil
	}
	return &Connected{
		Address: client.address,
		Server:  hex.EncodeToString(client.serverPublicKey),
	}
}

// String - return a string description of a client
func (client *clientData) String() string {
	return client.address
}

// GoString - return a basic information string for debugging purposes
func (client *clientData) GoString() string {
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
func (client *clientData) ServerPublicKey() []byte {
	return client.serverPublicKey
}

// internal poller called as go routine
// this cannot access clientData or a race condition will occur
func poller(monitor *zmq.Socket, sigReceive *zmq.Socket, queue chan<- Event) {

	poller := zmq.NewPoller()
	poller.Add(monitor, zmq.POLLIN)
	poller.Add(sigReceive, zmq.POLLIN)

loop:
	//log.Debug("start polling…")
	for {
		//log.Debug("waiting…")

		sockets, _ := poller.Poll(-1)
		for _, socket := range sockets {
			switch s := socket.Socket; s {
			case sigReceive:
				// receive the "stop" message
				_, err := sigReceive.RecvMessageBytes(0)
				if err != nil {
					logger.Panicf("poller: sigReceive error: %s", err)
				}

				break loop
			default:
				handleEvent(s, queue)
			}
		}
	}

	poller.RemoveBySocket(sigReceive)
	poller.RemoveBySocket(monitor)
	sigReceive.Close()
	monitor.Close()
	//log.Debug("stopped polling")
}

// process the socket events
func handleEvent(s *zmq.Socket, queue chan<- Event) error {
loop:
	for {
		ev, addr, v, err := s.RecvEvent(0)
		if zmq.Errno(syscall.EAGAIN) == zmq.AsErrno(err) {
			break loop
		}
		if err != nil {
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
	}
	return nil
}
