// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	subscriberSignal = "inproc://bitmark-subscriber-signal"
)

type subscriber struct {
	log     *logger.L
	push    *zmq.Socket
	pull    *zmq.Socket
	clients []*zmqutil.Client
	static  bool
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
	privateKey, err := zmqutil.ReadPrivateKeyFile(configuration.PrivateKey)
	if nil != err {
		return err
	}
	publicKey, err := zmqutil.ReadPublicKeyFile(configuration.PublicKey)
	if nil != err {
		return err
	}
	log.Tracef("server public:  %q", publicKey)
	log.Tracef("server private: %q", privateKey)

	// signalling channel
	subscribe.push, subscribe.pull, err = zmqutil.NewSignalPair(subscriberSignal)
	if nil != err {
		return err
	}

	errX := error(nil)

	staticClients := len(configuration.Connect)
	if 0 == staticClients {
		subscribe.clients = make([]*zmqutil.Client, dynamicClients)
		subscribe.static = false
		for i := 0; i < dynamicClients; i += 1 {
			client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, reqTimeout)
			if nil != err {
				log.Errorf("client[%d]  error: %v", i, err)
				errX = err
				goto fail
			}

			subscribe.clients[i] = client
		}
		return nil
	} else {
		subscribe.clients = make([]*zmqutil.Client, staticClients)
		subscribe.static = true
	}

	// initially connect all static sockets
	for i, c := range configuration.Subscribe {
		address := c.Address
		serverPublicKey, err := hex.DecodeString(c.PublicKey)
		if nil != err {
			log.Errorf("client[%d]=public: %q  error: %v", i, c.PublicKey, err)
			errX = err
			goto fail
		}

		client, err := zmqutil.NewClient(zmq.SUB, privateKey, publicKey, reqTimeout)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}

		subscribe.clients[i] = client

		err = client.Connect(address, serverPublicKey)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}
		log.Infof("public key: %x  at: %q", serverPublicKey, address)
	}

	return nil

	// error handling
fail:
	zmqutil.CloseClients(subscribe.clients)
	return errX
}

// wait for new blocks or new payment items
// to ensure the queue integrity as heap is not thread-safe
func (subscribe *subscriber) Run(args interface{}, shutdown <-chan struct{}) {

	log := subscribe.log

	log.Info("starting…")

	go func() {
		poller := zmq.NewPoller()
		for _, c := range subscribe.clients {
			if nil != c {
				c.Add(poller, zmq.POLLIN)
				// poller.Add(c.Socket(), zmq.POLLIN) // ***** FIX THIS: socket?
			}
		}
		poller.Add(subscribe.pull, zmq.POLLIN)
	loop:
		for {
			log.Info("waiting…")
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case subscribe.pull:
					s.Recv(0)
					break loop
				default:

					data, err := s.RecvMessageBytes(0)
					if nil != err {
						log.Errorf("receive error: %v", err)

					} else {
						subscribe.process(data)
					}
				}
			}
		}
		subscribe.pull.Close()
		zmqutil.CloseClients(subscribe.clients)
	}()

	// wait for shutdown
	<-shutdown
	subscribe.push.SendMessage("stop")
	subscribe.push.Close()
}

// process the received subscription
func (subscribe *subscriber) process(data [][]byte) {

	log := subscribe.log
	log.Info("incoming message")

	switch string(data[0]) {
	case "block":
		log.Infof("received block: %x", data[1])
	case "assets":
		log.Infof("received assets: %x", data[1])
	case "issues":
		log.Infof("received issues: %x", data[1])
	case "proof":
		log.Infof("received proof: %x", data[1])
	case "rpc":
		log.Infof("received rpc: %q fingerprint:%x", data[1], data[2])
	case "broadcast":
		log.Infof("received broadcast: %q", data[1])
	case "listener":
		log.Infof("received listener: %q", data[1])

	default:
		log.Warnf("received unhandled: %x", data)

	}
}
