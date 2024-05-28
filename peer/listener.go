// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"syscall"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	listenerZapDomain         = "listen"
	listenerSignal            = "inproc://bitmark-listener-signal"
	listenerIPv4MonitorSignal = "inproc://listener-ipv4-monitor-signal"
	listenerIPv6MonitorSignal = "inproc://listener-ipv6-monitor-signal"
)

type listener struct {
	log         *logger.L
	chain       string
	version     string      // server version
	sigSend     *zmq.Socket // signal send
	sigReceive  *zmq.Socket // signal receive
	socket4     *zmq.Socket // IPv4 traffic
	socket6     *zmq.Socket // IPv6 traffic
	monitor4    *zmq.Socket // IPv4 socket monitor
	monitor6    *zmq.Socket // IPv6 socket monitor
	connections uint64      // total incoming connections
}

// type to hold server info
type serverInfo struct {
	Version string `json:"version"`
	Chain   string `json:"chain"`
	Normal  bool   `json:"normal"`
	Height  uint64 `json:"height"`
}

// initialise the listener
func (lstn *listener) initialise(privateKey []byte, publicKey []byte, listen []string, version string) error {

	log := logger.New("listener")

	lstn.chain = mode.ChainName()
	lstn.log = log
	lstn.version = version
	lstn.connections = 0

	log.Info("initialising…")

	c, err := util.NewConnections(listen)
	if err != nil {
		log.Errorf("ip and port error: %s", err)
		return err
	}

	// signalling channel
	lstn.sigReceive, lstn.sigSend, err = zmqutil.NewSignalPair(listenerSignal)
	if err != nil {
		return err
	}

	// allocate IPv4 and IPv6 sockets
	lstn.socket4, lstn.socket6, err = zmqutil.NewBind(log, zmq.REP, listenerZapDomain, privateKey, publicKey, c)
	if err != nil {
		log.Errorf("bind error: %s", err)
		return err
	}

	if lstn.socket4 != nil {
		lstn.monitor4, err = zmqutil.NewMonitor(lstn.socket4, listenerIPv4MonitorSignal, zmq.EVENT_ALL)
		if err != nil {
			return err
		}
	}

	if lstn.socket6 != nil {
		lstn.monitor6, err = zmqutil.NewMonitor(lstn.socket6, listenerIPv6MonitorSignal, zmq.EVENT_ALL)
		if err != nil {
			return err
		}
	}

	return nil
}

// wait for incoming requests, process them and reply
func (lstn *listener) Run(args interface{}, shutdown <-chan struct{}) {

	log := lstn.log

	log.Info("starting…")

	go func() {
		poller := zmq.NewPoller()
		if lstn.socket4 != nil {
			poller.Add(lstn.socket4, zmq.POLLIN)
			poller.Add(lstn.monitor4, zmq.POLLIN)
		}
		if lstn.socket6 != nil {
			poller.Add(lstn.socket6, zmq.POLLIN)
			poller.Add(lstn.monitor6, zmq.POLLIN)
		}
		poller.Add(lstn.sigReceive, zmq.POLLIN)
	loop:
		for {
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case lstn.socket4:
					lstn.process(lstn.socket4)
				case lstn.socket6:
					lstn.process(lstn.socket6)
				case lstn.sigReceive:
					s.RecvMessageBytes(0)
					break loop
				case lstn.monitor4:
					lstn.handleEvent(lstn.monitor4)
				case lstn.monitor6:
					lstn.handleEvent(lstn.monitor6)
				}
			}
		}
		log.Info("shutting down")
		lstn.sigReceive.Close()
		if lstn.socket4 != nil {
			lstn.socket4.Close()
		}
		if lstn.socket6 != nil {
			lstn.socket6.Close()
		}
		log.Info("stopped")
	}()

	// wait for shutdown
	log.Debug("waiting…")
	<-shutdown
	log.Info("initiate shutdown")
	lstn.sigSend.SendMessage("stop")
	lstn.sigSend.Close()
}

// process the listen and return response to client
func (lstn *listener) process(socket *zmq.Socket) {

	log := lstn.log

	log.Debug("process starting…")
	for i := 0; lstn.processOne(i, socket); i += 1 {
	}
}

func (lstn *listener) processOne(i int, socket *zmq.Socket) bool {
	log := lstn.log

	data, err := socket.RecvMessageBytes(zmq.DONTWAIT)
	if zmq.Errno(syscall.EAGAIN) == zmq.AsErrno(err) {
		lstn.log.Infof("processed: %d events", i)
		return false
	}
	if err != nil {
		log.Errorf("receive: %d error: %s", i, err)
		return false
	}

	if len(data) < 2 {
		listenerSendError(socket, fmt.Errorf("packet too short"))
		return true
	}

	theChain := string(data[0])
	if theChain != lstn.chain {
		log.Errorf("invalid chain: actual: %q  expect: %s", theChain, lstn.chain)
		listenerSendError(socket, fmt.Errorf("invalid chain: actual: %q  expect: %s", theChain, lstn.chain))
		return true
	}

	fn := string(data[1])
	parameters := data[2:]

	log.Debugf("received message: %q: %x", fn, parameters)

	result := []byte{}

	switch fn {

	case "I": // server information
		info := serverInfo{
			Version: lstn.version,
			Chain:   mode.ChainName(),
			Normal:  mode.Is(mode.Normal),
			Height:  blockheader.Height(),
		}
		result, err = json.Marshal(info)
		logger.PanicIfError("JSON encode error: %s", err)

	case "N": // get block number
		blockNumber := blockheader.Height()
		result = make([]byte, 8)
		binary.BigEndian.PutUint64(result, blockNumber)

	case "B": // get packed block
		if len(parameters) != 1 {
			err = fault.MissingParameters
		} else if len(parameters[0]) == 8 {
			result = storage.Pool.Blocks.Get(parameters[0])
			if result == nil {
				err = fault.BlockNotFound
			}
		} else {
			err = fault.BlockNotFound
		}

	case "H": // get block hash
		if len(parameters) != 1 {
			err = fault.MissingParameters
		} else if len(parameters[0]) == 8 {
			number := binary.BigEndian.Uint64(parameters[0])
			d, e := blockheader.DigestForBlock(number)
			if e == nil {
				result = d[:]
			} else {
				err = e
			}
		} else {
			err = fault.BlockNotFound
		}

	case "R": // registration: chain, publicKey, listeners, timestamp
		if len(parameters) < 4 {
			listenerSendError(socket, fault.MissingParameters)
			return true
		}
		chain := mode.ChainName()
		if string(parameters[0]) != chain {
			listenerSendError(socket, fault.IncorrectChain)
			return true
		}

		timestamp := binary.BigEndian.Uint64(parameters[3])
		announce.AddPeer(parameters[1], parameters[2], timestamp) // publicKey, listeners, timestamp
		publicKey, listeners, ts, err := announce.GetRandom(parameters[1])
		if err != nil {
			listenerSendError(socket, err)
			return true
		}

		var binTs [8]byte
		binary.BigEndian.PutUint64(binTs[:], uint64(ts.Unix()))

		_, err = socket.Send(fn, zmq.SNDMORE)
		logger.PanicIfError("Listener", err)
		_, err = socket.Send(chain, zmq.SNDMORE)
		logger.PanicIfError("Listener", err)
		_, err = socket.SendBytes(publicKey, zmq.SNDMORE)
		logger.PanicIfError("Listener", err)
		_, err = socket.SendBytes(listeners, zmq.SNDMORE)
		logger.PanicIfError("Listener", err)
		_, err = socket.SendBytes(binTs[:], 0)
		logger.PanicIfError("Listener", err)

		return true

	default: // other commands as subscription-type commands
		processSubscription(log, fn, parameters)
		result = []byte{'A'}
	}

	if err != nil {
		listenerSendError(socket, err)
		return true
	}

	// send results
	_, err = socket.Send(fn, zmq.SNDMORE)
	logger.PanicIfError("Listener", err)
	_, err = socket.SendBytes(result, 0)
	logger.PanicIfError("Listener", err)

	log.Infof("sent: %q  result: %x", fn, result)
	return true
}

// process the socket events
func (lstn *listener) handleEvent(socket *zmq.Socket) {

loop:
	for i := 0; true; i++ {
		ev, addr, v, err := socket.RecvEvent(zmq.DONTWAIT)
		if zmq.Errno(syscall.EAGAIN) == zmq.AsErrno(err) {
			lstn.log.Infof("received: %d events", i)
			break loop
		}
		if err != nil {
			lstn.log.Errorf("receive event: %d error: %s", i, err)
			return
		}
		lstn.log.Debugf("event: %q  address: %q  value: %d", ev, addr, v)

		switch ev {
		case zmq.EVENT_ACCEPTED:
			lstn.connections += 1
		case zmq.EVENT_DISCONNECTED:
			if lstn.connections > 0 {
				lstn.connections -= 1
			}
		default:
		}
	}
}

// send an error packet
func listenerSendError(socket *zmq.Socket, err error) {
	errorMessage := err.Error()
	_, err = socket.Send("E", zmq.SNDMORE)
	logger.PanicIfError("Listener", err)
	_, err = socket.Send(errorMessage, 0)
	logger.PanicIfError("Listener", err)
}
