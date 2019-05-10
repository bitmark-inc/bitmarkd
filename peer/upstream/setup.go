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
	queueSize     = 50 // 0 => synchronous queue
)

// Upstream - structure to hold an upstream connection
type Upstream struct {
	sync.RWMutex
	log         *logger.L
	client      *zmqutil.Client
	registered  bool
	blockHeight uint64
	shutdown    chan<- struct{}
}

// atomically incremented counter for log names
var upstreamCounter counter.Counter

// New - create a connection to an upstream server
func New(privateKey []byte, publicKey []byte, timeout time.Duration) (*Upstream, error) {
	client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, timeout)
	if nil != err {
		return nil, err
	}

	n := upstreamCounter.Increment()

	shutdown := make(chan struct{})
	u := &Upstream{
		log:         logger.New(fmt.Sprintf("upstream@%d", n)),
		client:      client,
		registered:  false,
		blockHeight: 0,
		shutdown:    shutdown,
	}
	go upstreamRunner(u, shutdown)
	return u, nil
}

// Destroy - shutdown a connection
func (u *Upstream) Destroy() {
	if nil != u {
		close(u.shutdown)
	}
}

// ResetServer - clear Server side info of Zmq client for reusing the
// upstream
func (u *Upstream) ResetServer() {
	u.GetClient().ResetServer()
	u.registered = false
	u.blockHeight = 0
}

// IsConnectedTo - check the current destination
//
// does not mean actually connected, as could be in a timeout and
// reconnect state
func (u *Upstream) IsConnectedTo(serverPublicKey []byte) bool {
	return u.client.IsConnectedTo(serverPublicKey)
}

// IsOK - check if registered and have a valid connection
func (u *Upstream) IsOK() bool {
	return u.registered
}

// ConnectedTo - if registered return the connection data
func (u *Upstream) ConnectedTo() *zmqutil.Connected {
	return u.client.ConnectedTo()
}

// Connect - connect (or reconnect) to a specific server
func (u *Upstream) Connect(address *util.Connection, serverPublicKey []byte) error {
	u.log.Infof("connecting to address: %s", address)
	u.log.Infof("connecting to server: %x", serverPublicKey)
	u.Lock()
	err := u.client.Connect(address, serverPublicKey, mode.ChainName())
	if nil == err {
		err = register(u.client, u.log)
	}
	u.Unlock()
	return err
}

// GetClient - return the internal ZeroMQ client data
func (u *Upstream) GetClient() *zmqutil.Client {
	return u.client
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

// Ping - ping to check the connection
func (u *Upstream) Ping() (success bool) {

	// critical section - lock out the runner process
	u.Lock()

	var data [][]byte
	err := u.client.Send("P")
	if nil == err {
		data, err = u.client.Receive(0)
	}

	u.Unlock()

	if nil != err {
		u.log.Errorf("ping to server %s failed with error %s", u.client, err)
		return
	}

	if 0 == len(data) {
		return
	}

	switch string(data[0]) {
	case "P":
		// Ping to peer successfully
		u.log.Infof("ping to server %s success", u.client)
		success = true
	default:
	}
	return
}

// loop to handle upstream communication
func upstreamRunner(u *Upstream, shutdown <-chan struct{}) {
	log := u.log

	log.Debug("starting…")

	queue := messagebus.Bus.Broadcast.Chan(queueSize)

	timer := time.After(cycleInterval)

loop:
	for {
		log.Debug("waiting…")

		select {
		case <-shutdown:
			break loop

		case <-timer:
			timer = time.After(cycleInterval)
			u.Lock()
			if !u.registered {
				err := register(u.client, u.log)
				if fault.ErrNotConnected == err {
					log.Infof("register: %s", err)
					u.Unlock()
					continue loop // try again later
				} else if nil != err {
					log.Warnf("register: serverKey: %x register error: %s  ", u.GetClient().GetServerPublicKey(), err)
					err := u.client.Reconnect()
					if nil != err {
						log.Errorf("register: reconnect error: %s", err)
					}
					u.Unlock()
					continue loop // try again later
				}
				u.registered = true
			}

			h, err := getHeight(u.client, u.log)
			if nil == err {
				u.blockHeight = h
			} else {
				u.registered = false
				log.Errorf("getHeight: error: %s", err)
				err := u.client.Reconnect()
				if nil != err {
					log.Errorf("highestBlock: reconnect error: %s", err)
				}
			}
			u.Unlock()

		case item := <-queue:
			log.Debugf("from queue: %q  %x", item.Command, item.Parameters)

			if u.registered {
				u.Lock()
				err := push(u.client, u.log, &item)
				if nil != err {
					log.Errorf("push: error: %s", err)
					err := u.client.Reconnect()
					if nil != err {
						log.Errorf("push: reconnect error: %s", err)
					}
				}
				u.Unlock()
			}
		}
	}
	log.Info("shutting down…")
	u.client.Close()
	log.Info("stopped")
}

// register with server and check chain information
func register(client *zmqutil.Client, log *logger.L) error {

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
		return fmt.Errorf("register error: %q", data[1])
	case "R":
		if len(data) < 5 {
			return fmt.Errorf("register response incorrect: %x", data)
		}
		chain := mode.ChainName()
		received := string(data[1])
		if received != chain {
			log.Criticalf("expected chain: %q but received: %q", chain, received)
			logger.Panicf("expected chain: %q but received: %q", chain, received)
		}

		timestamp := binary.BigEndian.Uint64(data[4])
		log.Infof("register replied: public key: %x:  listeners: %x  timestamp: %d", data[2], data[3], timestamp)
		announce.AddPeer(data[2], data[3], timestamp) // publicKey, broadcasts, listeners
		return nil
	default:
		return fmt.Errorf("rpc unexpected response: %q", data[0])
	}
}

// must have lock held before calling this
func getHeight(client *zmqutil.Client, log *logger.L) (uint64, error) {

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
