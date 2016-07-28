// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	broadcasterZapDomain = "broadcaster"
)

type broadcaster struct {
	log     *logger.L
	socket4 *zmq.Socket
	socket6 *zmq.Socket
}

// initialise the broadcaster
func (brdc *broadcaster) initialise(privateKey []byte, publicKey []byte, broadcast []string) error {

	log := logger.New("broadcaster")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	brdc.log = log

	log.Info("initialising…")

	c, err := util.NewConnections(broadcast)
	if nil != err {
		log.Errorf("ip and port error: %v", err)
		return err
	}

	// allocate IPv4 and IPv6 sockets
	brdc.socket4, brdc.socket6, err = zmqutil.NewBind(log, zmq.PUB, broadcasterZapDomain, privateKey, publicKey, c)
	if nil != err {
		log.Errorf("bind error: %v", err)
		return err
	}

	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (brdc *broadcaster) Run(args interface{}, shutdown <-chan struct{}) {

	log := brdc.log

	log.Info("starting…")

	queue := messagebus.Chan()

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			brdc.log.Infof("sending: %s  data: %x", item.Command, item.Parameters)
			brdc.process(brdc.socket4, &item)
			brdc.process(brdc.socket6, &item)
		}
	}
	if nil != brdc.socket4 {
		brdc.socket4.Close()
	}
	if nil != brdc.socket6 {
		brdc.socket6.Close()
	}
}

// process some items into a block and publish it
func (brdc *broadcaster) process(socket *zmq.Socket, item *messagebus.Message) {
	if nil == socket {
		return
	}

	_, err := socket.Send(item.Command, zmq.SNDMORE|zmq.DONTWAIT)
	fault.PanicIfError("broadcaster", err)
	last := len(item.Parameters) - 1
	for i, p := range item.Parameters {
		if i == last {
			_, err = socket.SendBytes(p, 0|zmq.DONTWAIT)
		} else {
			_, err = socket.SendBytes(p, zmq.SNDMORE|zmq.DONTWAIT)
		}
		fault.PanicIfError("broadcaster", err)
	}
}
