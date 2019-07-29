// SPDX-License-Identifier: ISC
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
	queue := messagebus.Bus.Broadcast.Chan(messagebus.Default)
	cycleTimer := time.After(cycleInterval)

loop:
	for {
		log.Debug("waiting…")

		select {
		case <-shutdown:
			break loop

		case <-cycleTimer:
			cycleTimer = time.After(cycleInterval)

			u.RLock()
			if u.connected {
				u.RUnlock()

				remoteHeight, err := u.height()
				if nil == err {
					u.lastResponseTime = time.Now()

					u.Lock()
					u.remoteHeight = remoteHeight
					u.Unlock()

					publicKey := u.client.ServerPublicKey()
					messagebus.Bus.Announce.Send("updatetime", publicKey)
				} else {
					log.Warnf("highest block error: %s", err)
				}

				localHeight := blockheader.Height()
				digest, err := u.RemoteDigestOfHeight(localHeight)
				if nil != err {
					log.Errorf("getBlockDigest error: %s", err)
					continue loop
				}
				u.Lock()
				u.localHeight = localHeight
				u.remoteDigestOfLocalHeight = digest
				u.Unlock()
			} else {
				u.RUnlock()
				log.Trace("upstream not connected")
			}

		case item := <-queue:
			log.Debugf("from queue: %q  %x", item.Command, item.Parameters)

			u.RLock()
			if u.connected {
				u.RUnlock()
				err := u.push(&item)
				if nil != err {
					log.Errorf("push: error: %s", err)
				}
			} else {
				u.RUnlock()
				log.Trace("upstream not connected")
			}
		}
	}
	log.Info("shutting down…")
	u.client.Close()
	log.Info("stopped")
}

// start polling the socket
//
// it should be called as a goroutine to avoid blocking
func (u *Upstream) poller(shutdown <-chan struct{}, event <-chan zmqutil.Event) {

	log := u.log

	log.Debug("start polling…")
	var disconnected bool // flag to check unexpected disconnection

loop:
	for {
		select {
		case <-shutdown:
			break loop
		case e := <-event:
			u.handleEvent(e, &disconnected)
		}
	}
	log.Debug("stopped polling")
}

// process the socket events
func (u *Upstream) handleEvent(event zmqutil.Event, disconnected *bool) {
	log := u.log

	switch event.Event {
	case zmqutil.EVENT_DISCONNECTED,
		zmqutil.EVENT_CLOSED,
		zmqutil.EVENT_CONNECT_RETRIED,
		zmqutil.EVENT_HANDSHAKE_FAILED_NO_DETAIL,
		zmqutil.EVENT_HANDSHAKE_FAILED_PROTOCOL,
		zmqutil.EVENT_HANDSHAKE_FAILED_AUTH:

		log.Warnf("socket %q is disconnected. event: %q", event.Address, event.Event)
		*disconnected = true

		u.Lock()
		u.connected = false
		u.Unlock()

	case zmqutil.EVENT_CONNECTED, zmqutil.EVENT_CONNECT_DELAYED, zmqutil.EVENT_HANDSHAKE_SUCCEEDED:
		log.Infof("socket %q is connected", event.Address)

		if *disconnected {
			// the socket is automatically recovered after disconnected by zmq is not useful.
			// the request by this socket always return error `resource temporarily unavailable`
			// try to close/open the socket makes the socket works as expectation.
			log.Infof("reconnecting to %q", event.Address)
			err := u.client.Reconnect()
			if nil != err {
				u.log.Warnf("reconnect error: %s", err)
				return
			}
			log.Infof("reconnect to %q successful", event.Address)
			*disconnected = false
		}

		err := u.requestConnect()
		if nil == err {
			u.Lock()
			u.connected = true
			u.Unlock()
		} else {
			u.log.Debugf("request peer connection error: %s", err)
		}
	default:
		log.Warnf("socket %q unhandled event: %q (0x%x) value: %d", event.Address, event.Event, int(event.Event), event.Value)
	}

}

// register with server and check chain information
func (u *Upstream) requestConnect() error {
	log := u.log
	client := u.client
	log.Debugf("register: client: %s", client)

	u.RLock()
	err := announce.SendRegistration(client, "R")
	if nil != err {
		u.RUnlock()
		log.Errorf("register: %s send error: %s", client, err)
		return err
	}
	data, err := client.Receive(0)
	u.RUnlock()

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

func (u *Upstream) height() (uint64, error) {
	log := u.log
	client := u.client
	log.Infof("getHeight: client: %s", client)

	u.RLock()
	err := client.Send("N")
	if nil != err {
		u.RUnlock()
		log.Errorf("getHeight: %s send error: %s", client, err)
		return 0, err
	}

	data, err := client.Receive(0)
	u.RUnlock()

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

func (u *Upstream) push(item *messagebus.Message) error {
	log := u.log
	client := u.client
	log.Infof("push: client: %s  %q %x", client, item.Command, item.Parameters)

	u.RLock()
	err := client.Send(item.Command, item.Parameters)
	if nil != err {
		u.RUnlock()
		log.Errorf("push: %s send error: %s", client, err)
		return err
	}

	data, err := client.Receive(0)
	u.RUnlock()

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
