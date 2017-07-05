// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package discovery

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	subStopSignal = "inproc://discovery-subscriber-signal"
)

type discoverer struct {
	log      *logger.L
	push     *zmq.Socket
	pull     *zmq.Socket
	sub      *zmq.Socket
	req      *zmq.Socket
	handlers map[string]currencyHandler
}

func NewDiscoverer(reqEndpoint, subEndpoint string) (*discoverer, error) {
	log := logger.New("discovery-subscriber")
	if log == nil {
		return nil, fault.ErrInvalidLoggerChannel
	}

	push, pull, err := zmqutil.NewSignalPair(subStopSignal)
	if err != nil {
		return nil, fault.ErrNoConnectionsAvailable
	}

	sub, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		return nil, fault.ErrNoConnectionsAvailable
	}
	sub.Connect(subEndpoint)
	sub.SetSubscribe("")

	req, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return nil, fault.ErrNoConnectionsAvailable
	}
	req.Connect(reqEndpoint)

	handlers := make(map[string]currencyHandler)
	for c := currency.First; c <= currency.Last; c++ {
		switch c {
		case currency.Bitcoin:
			handlers[c.String()] = &bitcoinHandler{logger.New(c.String())}
		case currency.Litecoin:
			handlers[c.String()] = &litecoinHandler{logger.New(c.String())}
		default: // only fails if new module not correctly installed
			logger.Panicf("missing payment initialiser for Currency: %s", c.String())
		}
	}

	return &discoverer{log, push, pull, sub, req, handlers}, nil
}

func (d *discoverer) Run(args interface{}, shutdown <-chan struct{}) {
	d.recover()

	go func() {
		poller := zmq.NewPoller()
		poller.Add(d.sub, zmq.POLLIN)
		poller.Add(d.pull, zmq.POLLIN)

	loop:
		for {
			polled, _ := poller.Poll(-1)

			for _, p := range polled {
				switch s := p.Socket; s {
				case d.pull:
					if _, err := s.RecvMessageBytes(0); err != nil {
						d.log.Errorf("pull receive error: %v", err)
						break loop
					}
					break loop

				default:
					msg, err := s.RecvMessageBytes(0)
					if err != nil {
						d.log.Errorf("sub receive error: %v", err)
					}

					d.process(msg)
				}
			}
		}

		d.pull.Close()
		d.sub.Close()

		d.log.Info("stopped")
	}()

	d.log.Info("started")

	<-shutdown

	d.log.Info("stopping")
	d.push.SendMessage("stop")
	d.push.Close()
	d.req.Close()
}

func (d *discoverer) recover() {
	originTime := time.Now().Add(-constants.ReservoirTimeout)

	for c := currency.First; c <= currency.Last; c++ {
		d.log.Infof("start to fetch possible %s txs since time at %d", c.String(), originTime.Unix())

		d.req.SendMessage(c.String(), originTime.Unix())
		msg, err := d.req.RecvMessageBytes(0)
		if err != nil {
			d.log.Errorf("failed to receive message: %v", err)
		}

		d.handlers[c.String()].recover(msg[1])
	}
}

func (d *discoverer) process(data [][]byte) {
	if len(data) != 2 {
		d.log.Errorf("invalid message: %v", data)
		return
	}

	currency := string(data[0])
	d.handlers[currency].processTx(data[1])
}
