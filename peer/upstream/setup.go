// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package upstream

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

const (
	cycleInterval = 30 * time.Second
)

// Upstream - structure to hold an upstream connection
type Upstream struct {
	sync.RWMutex

	log         *logger.L
	client      *zmqutil.Client
	connected   bool
	blockHeight uint64
	shutdown    chan<- struct{}
}

// atomically incremented counter for log names
var upstreamCounter counter.Counter

// New - create a connection to an upstream server
func New(privateKey []byte, publicKey []byte, timeout time.Duration) (*Upstream, error) {

	client, event, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, timeout, zmq.EVENT_ALL)
	if nil != err {
		return nil, err
	}

	n := upstreamCounter.Increment()

	shutdown := make(chan struct{})

	u := &Upstream{
		log:         logger.New(fmt.Sprintf("upstream@%d", n)),
		client:      client,
		connected:   false,
		blockHeight: 0,
		shutdown:    shutdown,
	}
	go u.runner(shutdown)
	go u.poller(shutdown, event)
	return u, nil
}

// Destroy - shutdown a connection and terminate its background processes
func (u *Upstream) Destroy() {
	if nil != u {
		close(u.shutdown)
	}
}

// ResetServer - clear Server side info of Zmq client for reusing the
// upstream
func (u *Upstream) ResetServer() {
	u.client.Close()
	u.connected = false
	u.blockHeight = 0
}

// IsConnectedTo - check the current destination
//
// does not mean actually connected, as could be in a timeout and
// reconnect state
func (u *Upstream) IsConnectedTo(serverPublicKey []byte) bool {
	return u.client.IsConnectedTo(serverPublicKey)
}

// IsConnected - check if registered and have a valid connection
func (u *Upstream) IsConnected() bool {
	return u.connected
}

// ConnectedTo - if registered return the connection data
func (u *Upstream) ConnectedTo() *zmqutil.Connected {
	return u.client.ConnectedTo()
}

// Connect - connect (or reconnect) to a specific server
func (u *Upstream) Connect(address *util.Connection, serverPublicKey []byte) error {
	u.log.Infof("connecting to address: %s", address)
	u.log.Infof("connecting to server: %x", serverPublicKey)

	err := u.client.Connect(address, serverPublicKey, mode.ChainName())
	if nil == err {
		err = requestConnect(u.client, u.log)
		if nil == err {
			u.Lock()
			u.connected = true
			u.Unlock()
		}
	}
	return err
}

// ServerPublicKey - return the internal ZeroMQ client data
func (u *Upstream) ServerPublicKey() []byte {
	return u.client.ServerPublicKey()
}

// GetHeight - fetch height from last polled value
func (u *Upstream) GetHeight() uint64 {
	return u.blockHeight
}

// GetBlockDigest - fetch block digest from a specific block number
func (u *Upstream) GetBlockDigest(blockNumber uint64) (blockdigest.Digest, error) {
	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("H", parameter)
	if nil == err {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if nil != err {
		return blockdigest.Digest{}, err
	}

	if 2 != len(data) {
		return blockdigest.Digest{}, fault.ErrInvalidPeerResponse
	}

	switch string(data[0]) {
	case "E":
		return blockdigest.Digest{}, fault.InvalidError(string(data[1]))
	case "H":
		d := blockdigest.Digest{}
		if blockdigest.Length == len(data[1]) {
			err := blockdigest.DigestFromBytes(&d, data[1])
			return d, err
		}
	default:
	}
	return blockdigest.Digest{}, fault.ErrInvalidPeerResponse
}

// GetBlockData - fetch block data from a specific block number
func (u *Upstream) GetBlockData(blockNumber uint64) ([]byte, error) {
	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("B", parameter)
	if nil == err {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if nil != err {
		return nil, err
	}

	if 2 != len(data) {
		return nil, fault.ErrInvalidPeerResponse
	}

	switch string(data[0]) {
	case "E":
		return nil, fault.InvalidError(string(data[1]))
	case "B":
		return data[1], nil
	default:
	}
	return nil, fault.ErrInvalidPeerResponse
}

// loop to handle upstream communication
func (u *Upstream) runner(shutdown <-chan struct{}) {
	log := u.log

	log.Debug("starting…")

	// use default queue size
	queue := messagebus.Bus.Broadcast.Chan(-1)
	cycleTimer := time.After(cycleInterval)

loop:
	for {
		log.Debug("waiting…")

		select {
		case <-shutdown:
			break loop

		case <-cycleTimer:
			cycleTimer = time.After(cycleInterval)

			u.Lock()
			clientConnected := u.client.IsConnected()
			u.log.Debugf("client socket connected: %t", clientConnected)

			if clientConnected {

				if !u.connected {
					err := requestConnect(u.client, u.log)
					if nil != err {
						log.Warnf("serverKey: %x connect error: %s  ", u.client.ServerPublicKey(), err)
						u.Unlock()
						continue loop // try again later
					}
					u.connected = true
				}

				h, err := height(u.client, u.log)
				if nil == err {
					u.blockHeight = h
					publicKey := u.client.ServerPublicKey()
					timestamp := make([]byte, 8)
					binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
					messagebus.Bus.Announce.Send("updatetime", publicKey, timestamp)

				} else {
					log.Errorf("highestBlock: reconnect error: %s", err)
				}

			}
			u.Unlock()

		case item := <-queue:
			log.Debugf("from queue: %q  %x", item.Command, item.Parameters)

			u.Lock()
			if u.connected {
				err := push(u.client, u.log, &item)
				if nil != err {
					log.Errorf("push: error: %s", err)
				}
			}
			u.Unlock()
		}
	}
	log.Info("shutting down…")
	u.client.Close()
	log.Info("stopped")
}

// register with server and check chain information
func requestConnect(client *zmqutil.Client, log *logger.L) error {

	log.Debugf("register: client: %s", client)

	err := announce.SendRegistration(client, "R")
	if nil != err {
		log.Errorf("register: %s send error: %s", client, err)
		return err
	}
	data, err := client.Receive(0)
	if nil != err {
		log.Errorf("register: %s receive error: %s", client, err)
		return err
	}

	if len(data) < 2 {
		return fmt.Errorf("register received: %d  expected at least: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return fmt.Errorf("connection refused. register error: %q", data[1])
	case "R":
		if len(data) < 5 {
			return fmt.Errorf("connection refused. register response incorrect: %x", data)
		}
		chain := mode.ChainName()
		received := string(data[1])
		if received != chain {
			log.Errorf("connection refused. Expected chain: %q but received: %q", chain, received)
			return fmt.Errorf("connection refused.  expected chain: %q but received: %q ", chain, received)
		}
		timestamp := binary.BigEndian.Uint64(data[4])
		log.Infof("connection establised. register replied: public key: %x:  listeners: %x  timestamp: %d", data[2], data[3], timestamp)
		announce.AddPeer(data[2], data[3], timestamp) // publicKey, broadcasts, listeners
		return nil
	default:
		return fmt.Errorf("connection refused. rpc unexpected response: %q", data[0])
	}
}

// must have lock held before calling this
func height(client *zmqutil.Client, log *logger.L) (uint64, error) {

	log.Infof("getHeight: client: %s", client)

	err := client.Send("N")
	if nil != err {
		log.Errorf("getHeight: %s send error: %s", client, err)
		return 0, err
	}

	data, err := client.Receive(0)
	if nil != err {
		log.Errorf("push: %s receive error: %s", client, err)
		return 0, err
	}
	if 2 != len(data) {
		return 0, fmt.Errorf("getHeight received: %d  expected: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return 0, fmt.Errorf("rpc error response: %q", data[1])
	case "N":
		if 8 != len(data[1]) {
			return 0, fmt.Errorf("highestBlock: rpc invalid response: %q", data[1])
		}
		height := binary.BigEndian.Uint64(data[1])
		log.Infof("height: %d", height)
		return height, nil
	default:
		return 0, fmt.Errorf("rpc unexpected response: %q", data[0])
	}
}

// must have lock held before calling this
func push(client *zmqutil.Client, log *logger.L, item *messagebus.Message) error {

	log.Infof("push: client: %s  %q %x", client, item.Command, item.Parameters)

	err := client.Send(item.Command, item.Parameters)
	if nil != err {
		log.Errorf("push: %s send error: %s", client, err)
		// Drop the message from cache for retrying later
		messagebus.Bus.Broadcast.DropCache(*item)
		return err
	}

	data, err := client.Receive(0)
	if nil != err {
		log.Errorf("push: %s receive error: %s", client, err)
		return err
	}
	if 2 != len(data) {
		return fmt.Errorf("push received: %d  expected: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return fmt.Errorf("rpc error response: %q", data[1])
	case item.Command:
		log.Debugf("push: client: %s complete: %q", client, data[1])
		return nil
	default:
		return fmt.Errorf("rpc unexpected response: %q", data[0])
	}
}

// start polling the socket
//
// it should be called as a goroutine to avoid blocking
func (u *Upstream) poller(shutdown <-chan struct{}, event <-chan zmqutil.Event) {

	log := u.log

	log.Debug("start polling…")

loop:
	for {
		select {
		case <-shutdown:
			break loop
		case e := <-event:
			u.handleEvent(e)
		}
	}
	log.Debug("stopped polling")
}

// process the socket events
func (u *Upstream) handleEvent(event zmqutil.Event) {

	u.log.Debugf("event: %q  address: %q  value: %d", event.Event, event.Address, event.Value)

	switch event.Event {
	case zmq.EVENT_DISCONNECTED:
		u.Lock()
		u.reconnect()
		u.Unlock()

	default:
	}
}

// reconnect to server
//
// need to hold the lock before calling
func (u *Upstream) reconnect() error {

	u.connected = false

	// try to reconnect
	u.log.Infof("reconnecting to [%s]…", u.client)
	err := u.client.Reconnect()
	if nil != err {
		u.log.Errorf("reconnect to [%s] error: %s", u.client, err)
		return err
	}

	u.log.Infof("reconnect to [%s] successfully", u.client)

	return nil
}
