// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package publish

import (
	"time"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

const (
	broadcasterZapDomain = "broadcaster"
	heartbeatInterval    = 60 * time.Second
	heartbeatTimeout     = 2 * heartbeatInterval
)

type broadcaster struct {
	log     *logger.L
	chain   string
	socket4 *zmq.Socket
	socket6 *zmq.Socket
}

// initialise the broadcaster
func (brdc *broadcaster) initialise(privateKey []byte, publicKey []byte, broadcast []string) error {

	log := logger.New("broadcaster")

	brdc.chain = mode.ChainName()
	brdc.log = log

	log.Info("initialising…")

	c, err := util.NewConnections(broadcast)
	if nil != err {
		log.Errorf("ip and port error: %s", err)
		return err
	}

	// allocate IPv4 and IPv6 sockets
	brdc.socket4, brdc.socket6, err = zmqutil.NewBind(log, zmq.PUB, broadcasterZapDomain, privateKey, publicKey, c)
	if nil != err {
		log.Errorf("bind error: %s", err)
		return err
	}

	return nil
}

// broadcasting main loop
func (brdc *broadcaster) Run(args interface{}, shutdown <-chan struct{}) {

	log := brdc.log

	log.Info("starting…")

	// use default queue size
	queue := messagebus.Bus.Broadcast.Chan(-1)

loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			log.Infof("sending: %s  data: %x", item.Command, item.Parameters)
			if nil == brdc.socket4 && nil == brdc.socket6 {
				log.Error("no IPv4 or IPv6 socket for broadcast")
			}
			if err := brdc.process(brdc.socket4, &item); nil != err {
				log.Criticalf("IPv4 error: %s", err)
				logger.Panicf("broadcaster: IPv4 error: %s", err)
			}
			if err := brdc.process(brdc.socket6, &item); nil != err {
				log.Criticalf("IPv6 error: %s", err)
				logger.Panicf("broadcaster: IPv6 error: %s", err)
			}

		case <-time.After(heartbeatInterval): // timeout on queue empty
			// this will only occur if so data was sent during the interval
			beat := &messagebus.Message{
				Command:    "heart",
				Parameters: [][]byte{[]byte("beat")},
			}
			log.Info("send heartbeat")

			if nil == brdc.socket4 && nil == brdc.socket6 {
				log.Error("no IPv4 or IPv6 socket for heartbeat")
			}
			if err := brdc.process(brdc.socket4, beat); nil != err {
				log.Criticalf("IPv4 error: %s", err)
				logger.Panicf("broadcaster: IPv4 error: %s", err)
			}
			if err := brdc.process(brdc.socket6, beat); nil != err {
				log.Criticalf("IPv6 error: %s", err)
				logger.Panicf("broadcaster: IPv6 error: %s", err)
			}
		}
	}
	log.Info("shutting down…")
	if nil != brdc.socket4 {
		brdc.socket4.Close()
	}
	if nil != brdc.socket6 {
		brdc.socket6.Close()
	}
	log.Info("stopped")
}

// process some items into a block and publish it
func (brdc *broadcaster) process(socket *zmq.Socket, item *messagebus.Message) error {
	if nil == socket {
		return nil
	}

	_, err := socket.Send(brdc.chain, zmq.SNDMORE|zmq.DONTWAIT)
	if nil != err {
		// Drop the message from cache for retrying later
		messagebus.DropCache(*item)
		return err
	}

	_, err = socket.Send(item.Command, zmq.SNDMORE|zmq.DONTWAIT)
	if nil != err {
		// Drop the message from cache for retrying later
		messagebus.DropCache(*item)
		return err
	}

	last := len(item.Parameters) - 1
	for i, p := range item.Parameters {
		if i == last {
			_, err = socket.SendBytes(p, 0|zmq.DONTWAIT)
		} else {
			_, err = socket.SendBytes(p, zmq.SNDMORE|zmq.DONTWAIT)
		}
		if nil != err {
			// Drop the message from cache for retrying later
			messagebus.DropCache(*item)
			return err
		}
	}
	return nil
}
