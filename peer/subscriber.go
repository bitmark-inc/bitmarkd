// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	subscriberSignal = "inproc://bitmark-subscriber-signal"
)

type subscriber struct {
	log    *logger.L
	push   *zmq.Socket
	pull   *zmq.Socket
	socket *zmq.Socket
}

// initialise the subscriber
func (subscribe *subscriber) initialise(configuration *Configuration) error {

	log := logger.New("subscriber")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	subscribe.log = log

	log.Info("initialising…")

	// read the keys
	privateKey, err := zmqutil.ReadKeyFile(configuration.PrivateKey)
	if nil != err {
		return err
	}
	publicKey, err := zmqutil.ReadKeyFile(configuration.PublicKey)
	if nil != err {
		return err
	}

	// send half signalling channel
	push, err := zmq.NewSocket(zmq.PUSH)
	if nil != err {
		return err
	}
	push.SetLinger(0)
	err = push.Bind(subscriberSignal)
	if nil != err {
		return err
	}

	subscribe.push = push

	// receive half of signalling channel
	pull, err := zmq.NewSocket(zmq.PULL)
	if nil != err {
		return err
	}
	pull.SetLinger(0)
	err = pull.Connect(subscriberSignal)
	if nil != err {
		return err
	}

	subscribe.pull = pull

	socket, err := zmq.NewSocket(zmq.SUB)
	if nil != err {
		return err
	}
	subscribe.socket = socket

	// setup as client
	socket.SetCurveServer(0)
	socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(privateKey)
	log.Tracef("server public:  %q", publicKey)
	log.Tracef("server private: %q", privateKey)

	socket.SetIdentity(publicKey) // just use public key for identity

	// set subscription prefix - empty => receive everything
	socket.SetSubscribe("")

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
	for i, c := range configuration.Subscribe {
		address := c.Address
		serverPublicKey := c.PublicKey
		socket.SetCurveServerkey(serverPublicKey)
		log.Tracef("server public: %q", serverPublicKey)

		connectTo, err := util.CanonicalIPandPort("tcp://", address)
		if nil != err {
			log.Errorf("subscriber[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}

		err = socket.Connect(connectTo)
		if nil != err {
			log.Errorf("subscribe[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}
		log.Infof("subscribe to: %q", address)
	}
	return nil
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (subscribe *subscriber) Run(args interface{}, shutdown <-chan struct{}) {

	log := subscribe.log

	log.Info("starting…")

	go func() {
		poller := zmq.NewPoller()
		poller.Add(subscribe.socket, zmq.POLLIN)
		poller.Add(subscribe.pull, zmq.POLLIN)
	loop:
		for {
			log.Info("waiting…")
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case subscribe.socket:
					subscribe.process()
				case subscribe.pull:
					s.Recv(0)
					break loop
				}
			}
		}
		subscribe.pull.Close()
		subscribe.socket.Close()
	}()

	// wait for shutdown
	<-shutdown
	subscribe.push.SendMessage("stop")
	subscribe.push.Close()
}

// process the received subscription
func (subscribe *subscriber) process() {

	log := subscribe.log
	log.Info("incoming message")

	data, err := subscribe.socket.RecvMessageBytes(0)
	if nil != err {
		log.Errorf("receive error: %v", err)
		return
	}

	log.Infof("received message: %x", data)

}
