// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"github.com/bitmark-inc/bitmarkd/constants"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"sync"
	"time"
)

const (
	discovererStopSignal = "inproc://discoverer-stop-signal"

	blockchainCheckInterval = 60 * time.Second
)

// discoverer listens to discovery proxy to get the possible txs
type discoverer struct {
	log  *logger.L
	push *zmq.Socket
	pull *zmq.Socket
	sub  *zmq.Socket
	req  *zmq.Socket
}

func newDiscoverer(subHostPort, reqHostPort string) (*discoverer, error) {

	log := logger.New("discoverer")

	subConnection, err := util.NewConnection(subHostPort)
	if err != nil {
		log.Errorf("invalid subscribe connection: %q expect IP4:port or [IP6]:port  error: %s", subHostPort, err)
		return nil, err
	}
	reqConnection, err := util.NewConnection(reqHostPort)
	if err != nil {
		log.Errorf("request connection: %q expect IP4:port or [IP6]:port  error: %s", reqHostPort, err)
		return nil, err
	}

	push, pull, err := zmqutil.NewSignalPair(discovererStopSignal)
	if err != nil {
		return nil, err
	}

	sub, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		return nil, err
	}

	subAddr, subIPv6 := subConnection.CanonicalIPandPort("tcp://")
	err = sub.SetIpv6(subIPv6)
	if err != nil {
		return nil, err
	}

	err = sub.Connect(subAddr)
	if err != nil {
		return nil, err
	}

	sub.SetSubscribe("")

	log.Infof("subscribe to: %q  IPv6: %t", subAddr, subIPv6)

	req, err := zmq.NewSocket(zmq.REQ)
	if err != nil {
		return nil, err
	}

	reqAddr, reqIPv6 := reqConnection.CanonicalIPandPort("tcp://")
	err = req.SetIpv6(reqIPv6)
	if err != nil {
		return nil, err
	}
	err = req.Connect(reqAddr)
	if err != nil {
		return nil, err
	}

	log.Infof("connect to: %q  IPv6: %t", reqAddr, reqIPv6)

	disc := &discoverer{
		log:  log,
		push: push,
		pull: pull,
		sub:  sub,
		req:  req,
	}
	return disc, nil
}

func (d *discoverer) Run(args interface{}, shutdown <-chan struct{}) {

	d.log.Info("starting…")

	d.retrievePastTxs()

	go func() {
		poller := zmq.NewPoller()
		poller.Add(d.sub, zmq.POLLIN)
		poller.Add(d.pull, zmq.POLLIN)

	loop:
		for {
			polled, _ := poller.Poll(-1)

			// TODO: add hearbeat
			for _, p := range polled {
				switch s := p.Socket; s {
				case d.pull:
					if _, err := s.RecvMessageBytes(0); err != nil {
						d.log.Errorf("pull receive error: %s", err)
						break loop
					}
					break loop

				default:
					msg, err := s.RecvMessageBytes(0)
					if err != nil {
						d.log.Errorf("sub receive error: %s", err)
					}

					d.assignHandler(msg)
				}
			}
		}

		d.pull.Close()
		d.sub.Close()

		d.log.Info("stopped")
	}()

	d.log.Info("started")

	<-shutdown

	d.push.SendMessage("stop")
	d.push.Close()
	d.req.Close()
}

func (d *discoverer) retrievePastTxs() {
	originTime := time.Now().Add(-constants.ReservoirTimeout)

	for currency, handler := range globalData.handlers {
		d.log.Infof("start to fetch possible %s txs since time at %d", currency, originTime.Unix())

		d.req.SendMessage(currency, originTime.Unix())
		msg, err := d.req.RecvMessageBytes(0)
		if err != nil {
			d.log.Errorf("failed to receive message: %v", err)
		}

		handler.processPastTxs(msg[1])
	}
}

func (d *discoverer) assignHandler(data [][]byte) {
	if len(data) != 2 {
		d.log.Errorf("invalid message: %v", data)
		return
	}

	currency := string(data[0])
	globalData.handlers[currency].processIncomingTx(data[1])
}

// checker periodically extracts possible txs in the latest block
type checker struct {
	log *logger.L
}

func (c *checker) Run(args interface{}, shutdown <-chan struct{}) {
	log := logger.New("checker")
	c.log = log

	log.Info("starting…")
loop:
	for {
		log.Info("begin…")
		select {
		case <-shutdown:
			break loop

		case <-time.After(blockchainCheckInterval):
			log.Info("checking…")
			var wg sync.WaitGroup
			for _, handler := range globalData.handlers {
				wg.Add(1)
				go handler.checkLatestBlock(&wg)
			}
			log.Info("waiting…")
			wg.Wait()
		}
	}
}
