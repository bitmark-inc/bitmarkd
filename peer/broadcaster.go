// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
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
func (brd *broadcaster) initialise(configuration *Configuration) error {

	log := logger.New("broadcaster")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	brd.log = log

	log.Info("initialising…")

	// read the keys
	privateKey, err := zmqutil.ReadPrivateKeyFile(configuration.PrivateKey)
	if nil != err {
		log.Errorf("read private key file: %q  error: %v", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadPublicKeyFile(configuration.PublicKey)
	if nil != err {
		log.Errorf("read public key file: %q  error: %v", configuration.PublicKey, err)
		return err
	}
	log.Tracef("server public:  %q", publicKey)
	log.Tracef("server private: %q", privateKey)

	// allocate IPv4 and IPv6 sockets
	brd.socket4, brd.socket6, err = zmqutil.NewBind(log, zmq.PUB, broadcasterZapDomain, privateKey, publicKey, configuration.Broadcast)
	if nil != err {
		log.Errorf("bind error: %v", err)
		return err
	}

	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (brd *broadcaster) Run(args interface{}, shutdown <-chan struct{}) {

	log := brd.log

	log.Info("starting…")

	queue := messagebus.Chan()

loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			brd.log.Infof("sending: %s  data: %x", item.Command, item.Parameters)
			brd.process(brd.socket4, &item)
			brd.process(brd.socket6, &item)
		}
	}
	if nil != brd.socket4 {
		brd.socket4.Close()
	}
	if nil != brd.socket6 {
		brd.socket6.Close()
	}
}

// process some items into a block and publish it
func (brd *broadcaster) process(socket *zmq.Socket, item *messagebus.Message) {
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
