// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package publish

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	broadcasterZapDomain = "broadcaster"
	heartbeatInterval    = 60 * time.Second
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
	if err != nil {
		log.Errorf("ip and port error: %s", err)
		return err
	}

	// allocate IPv4 and IPv6 sockets
	brdc.socket4, brdc.socket6, err = zmqutil.NewBind(log, zmq.PUB, broadcasterZapDomain, privateKey, publicKey, c)
	if err != nil {
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
	queue := messagebus.Bus.Broadcast.Chan(messagebus.Default)

loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			log.Infof("sending: %s  data: %x", item.Command, item.Parameters)
			if brdc.socket4 == nil && brdc.socket6 == nil {
				log.Error("no IPv4 or IPv6 socket for broadcast")
			}
			if err := brdc.process(brdc.socket4, &item); err != nil {
				log.Criticalf("IPv4 error: %s", err)
				logger.Panicf("broadcaster: IPv4 error: %s", err)
			}
			if err := brdc.process(brdc.socket6, &item); err != nil {
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

			if brdc.socket4 == nil && brdc.socket6 == nil {
				log.Error("no IPv4 or IPv6 socket for heartbeat")
			}
			if err := brdc.process(brdc.socket4, beat); err != nil {
				log.Criticalf("IPv4 error: %s", err)
				logger.Panicf("broadcaster: IPv4 error: %s", err)
			}
			if err := brdc.process(brdc.socket6, beat); err != nil {
				log.Criticalf("IPv6 error: %s", err)
				logger.Panicf("broadcaster: IPv6 error: %s", err)
			}
		}
	}
	log.Info("shutting down…")
	if brdc.socket4 != nil {
		brdc.socket4.Close()
	}
	if brdc.socket6 != nil {
		brdc.socket6.Close()
	}
	log.Info("stopped")
}

// process some items into a block and publish it
func (brdc *broadcaster) process(socket *zmq.Socket, item *messagebus.Message) error {
	if socket == nil {
		return nil
	}

	_, err := socket.Send(brdc.chain, zmq.SNDMORE|zmq.DONTWAIT)
	if err != nil {
		return err
	}

	_, err = socket.Send(item.Command, zmq.SNDMORE|zmq.DONTWAIT)
	if err != nil {
		return err
	}

	last := len(item.Parameters) - 1
	for i, p := range item.Parameters {
		if i == last {
			_, err = socket.SendBytes(p, 0|zmq.DONTWAIT)
		} else {
			_, err = socket.SendBytes(p, zmq.SNDMORE|zmq.DONTWAIT)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
