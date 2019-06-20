// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package upstream

import (
	"encoding/binary"
	"fmt"
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

const (
	cycleInterval = 30 * time.Second
)

// atomically incremented counter for log names
var upstreamCounter counter.Counter

// New - create a connection to an upstream server
func New(privateKey []byte, publicKey []byte, timeout time.Duration) (UpstreamIntf, error) {

	client, event, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, timeout, zmq.EVENT_ALL)
	if nil != err {
		return nil, err
	}

	n := upstreamCounter.Increment()

	shutdown := make(chan struct{})
	upstreamStr := fmt.Sprintf("upstream@%d", n)
	u := &Upstream{
		name:      upstreamStr,
		log:       logger.New(upstreamStr),
		client:    client,
		connected: false,
		shutdown:  shutdown,
	}
	go u.runner(shutdown)
	go u.poller(shutdown, event)
	return u, nil
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
						log.Warnf("serverKey: %x connect error: %s  ", u.ServerPublicKey(), err)
						u.Unlock()
						continue loop // try again later
					}
					u.connected = true
				}

				remoteHeight, err := height(u.client, u.log)
				if nil == err {
					u.lastResponseTime = time.Now()
					u.remoteHeight = remoteHeight
					publicKey := u.ServerPublicKey()
					timestamp := make([]byte, 8)
					binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
					messagebus.Bus.Announce.Send("updatetime", publicKey, timestamp)
				} else {
					log.Errorf("highestBlock: reconnect error: %s", err)
				}
			} else if u.client.HasValidAddress() {
				u.reconnect()
			}
			u.Unlock()

			// XXX: need some refactor
			// GetBlockDigest has lock inside, so it cannot be put into
			// previous code block
			// two variables of clientConnected & u.connected seems to have similar
			// meaning, these two variables are not independent, e.g.
			// situation of clientConnected = false, u.connected = true should not exist
			if clientConnected && u.connected {
				localHeight := blockheader.Height()
				digest, err := u.RemoteDigestOfHeight(localHeight)
				if nil != err {
					log.Errorf("getBlockDigest error: %s", err)
					continue
				}
				u.Lock()
				u.localHeight = localHeight
				u.remoteDigestOfLocalHeight = digest
				u.Unlock()
			}

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
func requestConnect(client zmqutil.ClientIntf, log *logger.L) error {

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
		log.Infof("connection refused. register replied: public key: %x:  listeners: %x  timestamp: %d", data[2], data[3], timestamp)
		// publicKey, broadcasts, listeners
		announce.AddPeer(data[2], data[3], timestamp)
		return nil
	default:
		return fmt.Errorf("connection refused. rpc unexpected response: %q", data[0])
	}
}

// must have lock held before calling this
func height(client zmqutil.ClientIntf, log *logger.L) (uint64, error) {

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
func push(client zmqutil.ClientIntf, log *logger.L, item *messagebus.Message) error {

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
