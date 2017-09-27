// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"bytes"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"time"
)

const (
	subscriberSignal = "inproc://bitmark-subscriber-signal"
)

type subscriber struct {
	log          *logger.L
	chain        string
	push         *zmq.Socket
	pull         *zmq.Socket
	clients      []*zmqutil.Client
	dynamicStart int
}

// initialise the subscriber
func (sbsc *subscriber) initialise(privateKey []byte, publicKey []byte, subscribe []Connection, dynamicEnabled bool) error {

	log := logger.New("subscriber")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	sbsc.chain = mode.ChainName()
	sbsc.log = log

	log.Info("initialising…")

	// validate connection count
	staticCount := len(subscribe) // can be zero
	if 0 == staticCount && !dynamicEnabled {
		log.Error("zero static connections and dynamic is disabled")
		return fault.ErrNoConnectionsAvailable
	}

	// signalling channel
	err := error(nil)
	sbsc.push, sbsc.pull, err = zmqutil.NewSignalPair(subscriberSignal)
	if nil != err {
		return err
	}

	// all sockets
	sbsc.clients = make([]*zmqutil.Client, staticCount+offsetCount)
	sbsc.dynamicStart = staticCount // index of first dynamic socket
	globalData.subscriberClients = sbsc.clients

	// error for goto fail
	errX := error(nil)

	// initially connect all static sockets
	for i, c := range subscribe {
		address, err := util.NewConnection(c.Address)
		if nil != err {
			log.Errorf("client[%d]=address: %q  error: %v", i, c.Address, err)
			errX = err
			goto fail
		}
		serverPublicKey, err := hex.DecodeString(c.PublicKey)
		if nil != err {
			log.Errorf("client[%d]=public: %q  error: %v", i, c.PublicKey, err)
			errX = err
			goto fail
		}

		// prevent connection to self
		if bytes.Equal(publicKey, serverPublicKey) {
			errX = fault.ErrConnectingToSelfForbidden
			log.Errorf("client[%d]=public: %q  error: %v", i, c.PublicKey, errX)
			goto fail
		}

		client, err := zmqutil.NewClient(zmq.SUB, privateKey, publicKey, 0)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %v", i, c.Address, err)
			errX = err
			goto fail
		}

		sbsc.clients[i] = client

		err = client.Connect(address, serverPublicKey, sbsc.chain)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %v", i, c.Address, err)
			errX = err
			goto fail
		}
		log.Infof("public key: %x  at: %q", serverPublicKey, c.Address)
	}

	// just create sockets for dynamic clients
	for i := sbsc.dynamicStart; i < len(sbsc.clients); i += 1 {
		client, err := zmqutil.NewClient(zmq.SUB, privateKey, publicKey, 0)
		if nil != err {
			log.Errorf("client[%d]  error: %v", i, err)
			errX = err
			goto fail
		}

		sbsc.clients[i] = client
	}

	return nil

	// error handling
fail:
	zmqutil.CloseClients(sbsc.clients)
	return errX
}

// subscriber main loop
func (sbsc *subscriber) Run(args interface{}, shutdown <-chan struct{}) {

	log := sbsc.log

	log.Info("starting…")

	queue := messagebus.Bus.Subscriber.Chan()

	go func() {

		expiryRegister := make(map[*zmq.Socket]time.Time)
		checkAt := time.Now().Add(heartbeatTimeout)
		poller := zmqutil.NewPoller()

		for _, client := range sbsc.clients {
			socket := client.BeginPolling(poller, zmq.POLLIN)
			if nil != socket {
				expiryRegister[socket] = checkAt
			}
		}
		poller.Add(sbsc.pull, zmq.POLLIN)

	loop:
		for {
			log.Info("waiting…")

			//polled, _ := poller.Poll(-1)
			polled, _ := poller.Poll(heartbeatTimeout)

			now := time.Now()
			expiresAt := now.Add(heartbeatTimeout)
			if now.After(checkAt) {
				checkAt = expiresAt
				for s, expires := range expiryRegister {
					if now.After(expires) {
						client := zmqutil.ClientFromSocket(s)
						if nil == client { // this socket has been closed
							delete(expiryRegister, s)
						} else if client.IsConnected() {
							log.Warnf("reconnecting to: %q", client)
							skt, err := client.ReconnectReturningSocket()
							if nil != err {
								log.Errorf("reconnect error: %s", err)
							} else {
								delete(expiryRegister, s)
								// note this new entry may or may not be rescanned by range in this loop
								// since it will have future time it will not be immediately deleted
								expiryRegister[skt] = expiresAt
							}
						} else {
							expiryRegister[s] = expiresAt
						}
					} else if expires.Before(checkAt) {
						checkAt = expires
					}
				}
			}

			for _, p := range polled {
				switch s := p.Socket; s {
				case sbsc.pull:
					data, err := s.RecvMessageBytes(0)
					if nil != err {
						log.Errorf("pull receive error: %v", err)
						break loop
					}

					switch string(data[0]) {
					case "connect":
						command := string(data[1])
						publicKey := data[2]
						broadcasts := data[3]
						connectToPublisher(sbsc.log, sbsc.chain, sbsc.clients, sbsc.dynamicStart, command, publicKey, broadcasts)
					default:
						break loop
					}
				default:
					data, err := s.RecvMessageBytes(0)
					if nil != err {
						log.Errorf("receive error: %v", err)
					} else {

						dataLength := len(data)
						if dataLength < 3 {
							log.Warnf("with too few data: %d items", dataLength)
							continue loop
						}
						theChain := string(data[0])
						if theChain != sbsc.chain {
							log.Errorf("invalid chain: actual: %q  expect: %s", theChain, sbsc.chain)
							continue loop
						}
						processSubscription(sbsc.log, string(data[1]), data[2:])
					}
					expiryRegister[s] = expiresAt
				}
			}
		}
		log.Info("shutting down…")
		sbsc.pull.Close()
		zmqutil.CloseClients(sbsc.clients)
		log.Info("stopped")
	}()

loop:
	for {
		log.Info("select…")

		select {
		// wait for shutdown
		case <-shutdown:
			break loop
		// wait for message
		case item := <-queue:
			sbsc.log.Infof("received: %s  public key: %x  connect: %x", item.Command, item.Parameters[0], item.Parameters[1])
			sbsc.push.SendMessage("connect", item.Command, item.Parameters[0], item.Parameters[1])
		}
	}

	log.Info("initiate shutdown")
	sbsc.push.SendMessage("stop")
	sbsc.push.Close()
	log.Info("finished")
}
