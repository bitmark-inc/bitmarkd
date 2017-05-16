// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitcoin

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	subscriberSignal = "inproc://bitcoin-subscriber-signal"
)

type zcSubscriber struct {
	log     *logger.L
	push    *zmq.Socket
	pull    *zmq.Socket
	clients []*zmq.Socket
}

// initialise the subscriber
func (sbsc *zcSubscriber) initialise(subscribe []string) error {

	log := logger.New("bitcoin-subscriber")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	sbsc.log = log

	log.Info("initialising…")

	// validate connection count
	count := len(subscribe)
	if 0 == count {
		log.Error("zero connections")
		return fault.ErrNoConnectionsAvailable
	}

	// signalling channel
	err := error(nil)
	sbsc.push, sbsc.pull, err = zmqutil.NewSignalPair(subscriberSignal)
	if nil != err {
		return err
	}

	// all sockets
	sbsc.clients = make([]*zmq.Socket, count)

	// error for goto fail
	errX := error(nil)

	// initially connect all static sockets
	for i, c := range subscribe {
		conn, err := util.NewConnection(c)
		if nil != err {
			log.Errorf("client[%d]=address: %q  error: %v", i, c, err)
			errX = err
			goto fail
		}

		client, err := zmq.NewSocket(zmq.SUB)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %v", i, c, err)
			errX = err
			goto fail
		}

		address, v6 := conn.CanonicalIPandPort("tcp://")

		// set IPv6 state before connect
		err = client.SetIpv6(v6)
		if nil != err {
			errX = err
			goto fail
		}
		client.Connect(address)
		client.SetSubscribe("rawtx")

		sbsc.clients[i] = client
		log.Infof("bitcoin[%d] subscription to: %q", i, c)
	}

	return nil

	// error handling
fail:
	for _, s := range sbsc.clients {
		s.Close()
	}
	return errX
}

// subscriber main loop
func (sbsc *zcSubscriber) Run(args interface{}, shutdown <-chan struct{}) {

	log := sbsc.log

	log.Info("starting…")

	go func() {

		poller := zmq.NewPoller()

		for _, client := range sbsc.clients {
			poller.Add(client, zmq.POLLIN)
		}
		poller.Add(sbsc.pull, zmq.POLLIN)

	loop:
		for {
			log.Debug("waiting…")

			polled, _ := poller.Poll(-1)

			for _, p := range polled {
				switch s := p.Socket; s {
				case sbsc.pull:
					_, err := s.RecvMessageBytes(0)
					if nil != err {
						log.Errorf("pull receive error: %v", err)
						break loop
					}
					break loop

				default:
					data, err := s.RecvMessageBytes(0)
					if nil != err {
						log.Errorf("receive error: %v", err)
					} else {
						sbsc.process(data)
					}
				}
			}
		}
		log.Info("shutting down…")
		sbsc.pull.Close()
		for _, s := range sbsc.clients {
			s.Close()
		}

		log.Info("stopped")
	}()

	// wait fr termination
	<-shutdown

	log.Info("initiate shutdown")
	sbsc.push.SendMessage("stop")
	sbsc.push.Close()
	log.Info("finished")
}

// process the received subscription
func (sbsc *zcSubscriber) process(data [][]byte) {

	log := sbsc.log

	if 2 != len(data) {
		log.Errorf("invalid message: %v", data)
		return
	}

	switch topic := string(data[0]); topic {
	case "rawtx":
		checkForPaymentTransaction(log, hex.EncodeToString(data[1]))

	default:
		log.Errorf("invalid topic: %q", topic)
	}
}
