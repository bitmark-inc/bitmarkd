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
	log    *logger.L
	socket *zmq.Socket
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
	privateKey, err := zmqutil.ReadKeyFile(configuration.PrivateKey)
	if nil != err {
		log.Errorf("read private key file: %q  error: %v", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadKeyFile(configuration.PublicKey)
	if nil != err {
		log.Errorf("read public key file: %q  error: %v", configuration.PublicKey, err)
		return err
	}

	socket, err := zmq.NewSocket(zmq.PUB)
	if nil != err {
		return err
	}
	brd.socket = socket

	// ***** FIX THIS ****
	// this allows any client to connect
	zmq.AuthAllow(broadcasterZapDomain, "127.0.0.1/8")
	zmq.AuthCurveAdd(broadcasterZapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(privateKey)
	log.Tracef("server public:  %q", publicKey)
	log.Tracef("server private: %q", privateKey)

	socket.SetZapDomain(broadcasterZapDomain)

	socket.SetIdentity(publicKey) // just use public key for identity

	// ***** FIX THIS ****
	// maybe need to change above line to specific keys later
	//   e.g. zmq.AuthCurveAdd(serverPublicKey, client1PublicKey)
	//        zmq.AuthCurveAdd(serverPublicKey, client2PublicKey)
	// perhaps as part of ConnectTo

	// // basic socket options
	// socket.SetIpv6(true) // ***** FIX THIS find fix for FreeBSD libzmq4 ****
	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer
	for i, address := range configuration.Broadcast {
		bindTo, err := util.CanonicalIPandPort("tcp://", address)
		if nil != err {
			log.Errorf("broadcast[%d]=%q  error: %v", i, address, err)
			return err
		}

		err = socket.Bind(bindTo)
		if nil != err {
			log.Errorf("bind[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}
		log.Infof("publish on: %q", address)
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
			brd.process(&item)
		}
	}
	brd.socket.Close()
}

// process some items into a block and publish it
func (brd *broadcaster) process(item *messagebus.Message) {

	brd.log.Infof("sending: %s  data: %x", item.Kind, item.Data)

	// ***** FIX THIS: is the DONTWAIT flag needed or not?
	_, err := brd.socket.Send(item.Kind, zmq.SNDMORE|zmq.DONTWAIT)
	fault.PanicIfError("broadcaster", err)
	_, err = brd.socket.SendBytes(item.Data, 0|zmq.DONTWAIT)
	fault.PanicIfError("broadcaster", err)
}
