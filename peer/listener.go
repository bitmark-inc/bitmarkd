// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	zmq "github.com/pebbe/zmq4"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

const (
	listenerZapDomain = "listen"
	listenerSignal    = "inproc://bitmark-listener-signal"
)

type listener struct {
	log     *logger.L
	chain   string
	version string      // server version
	push    *zmq.Socket // signal send
	pull    *zmq.Socket // signal receive
	socket4 *zmq.Socket // IPv4 traffic
	socket6 *zmq.Socket // IPv6 traffic
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

	log.Info("initialising…")

	c, err := util.NewConnections(listen)
	if nil != err {
		log.Errorf("ip and port error: %s", err)
		return err
	}

	// signalling channel
	lstn.push, lstn.pull, err = zmqutil.NewSignalPair(listenerSignal)
	if nil != err {
		return err
	}

	// allocate IPv4 and IPv6 sockets
	lstn.socket4, lstn.socket6, err = zmqutil.NewBind(log, zmq.REP, listenerZapDomain, privateKey, publicKey, c)
	if nil != err {
		log.Errorf("bind error: %s", err)
		return err
	}

	return nil
}

// wait for incoming requests, process them and reply
func (lstn *listener) Run(args interface{}, shutdown <-chan struct{}) {

	log := lstn.log

	log.Info("starting…")

	go func() {
		poller := zmq.NewPoller()
		if nil != lstn.socket4 {
			poller.Add(lstn.socket4, zmq.POLLIN)
		}
		if nil != lstn.socket6 {
			poller.Add(lstn.socket6, zmq.POLLIN)
		}
		poller.Add(lstn.pull, zmq.POLLIN)
	loop:
		for {
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case lstn.socket4:
					lstn.process(lstn.socket4)
				case lstn.socket6:
					lstn.process(lstn.socket6)
				case lstn.pull:
					s.RecvMessageBytes(0)
					break loop
				}
			}
		}
		log.Info("shutting down")
		lstn.pull.Close()
		if nil != lstn.socket4 {
			lstn.socket4.Close()
		}
		if nil != lstn.socket6 {
			lstn.socket6.Close()
		}
		log.Info("stopped")
	}()

	// wait for shutdown
	log.Debug("waiting…")
	<-shutdown
	log.Info("initiate shutdown")
	lstn.push.SendMessage("stop")
	lstn.push.Close()
}

// process the listen and return response to client
func (lstn *listener) process(socket *zmq.Socket) {

	log := lstn.log

	log.Debug("process starting…")

	data, err := socket.RecvMessageBytes(0)
	if nil != err {
		log.Errorf("receive error: %s", err)
		return
	}

	if len(data) < 2 {
		listenerSendError(socket, fmt.Errorf("packet too short"))
		return
	}

	theChain := string(data[0])
	if theChain != lstn.chain {
		log.Errorf("invalid chain: actual: %q  expect: %s", theChain, lstn.chain)
		listenerSendError(socket, fmt.Errorf("invalid chain: actual: %q  expect: %s", theChain, lstn.chain))
		return
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
			Height:  block.GetHeight(),
		}
		result, err = json.Marshal(info)
		logger.PanicIfError("JSON encode error: %s", err)

	case "N": // get block number
		blockNumber := block.GetHeight()
		result = make([]byte, 8)
		binary.BigEndian.PutUint64(result, blockNumber)

	case "B": // get packed block
		if 1 != len(parameters) {
			err = fault.ErrMissingParameters
		} else if 8 == len(parameters[0]) {
			result = storage.Pool.Blocks.Get(parameters[0])
			if nil == result {
				err = fault.ErrBlockNotFound
			}
		} else {
			err = fault.ErrBlockNotFound
		}

	case "H": // get block hash
		if 1 != len(parameters) {
			err = fault.ErrMissingParameters
		} else if 8 == len(parameters[0]) {
			number := binary.BigEndian.Uint64(parameters[0])
			d, e := block.DigestForBlock(number)
			if nil == e {
				result = d[:]
			} else {
				err = e
			}
		} else {
			err = fault.ErrBlockNotFound
		}

	case "R": // registration: chain, publicKey, listeners, timestamp
		if len(parameters) < 4 {
			listenerSendError(socket, fault.ErrMissingParameters)
			return
		}
		chain := mode.ChainName()
		if string(parameters[0]) != chain {
			listenerSendError(socket, fault.ErrIncorrectChain)
			return
		}

		timestamp := binary.BigEndian.Uint64(parameters[3])
		announce.AddPeer(parameters[1], parameters[2], timestamp) // publicKey, listeners, timestamp
		publicKey, listeners, ts, err := announce.GetNext(parameters[1])
		if nil != err {
			listenerSendError(socket, err)
			return
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

		return

	default: // other commands as subscription-type commands
		processSubscription(log, fn, parameters)
		result = []byte{'A'}
	}

	if nil != err {
		listenerSendError(socket, err)
		return
	}

	// send results
	_, err = socket.Send(fn, zmq.SNDMORE)
	logger.PanicIfError("Listener", err)
	_, err = socket.SendBytes(result, 0)
	logger.PanicIfError("Listener", err)

	log.Infof("sent: %q  result: %x", fn, result)
}

// send an error packet
func listenerSendError(socket *zmq.Socket, err error) {
	errorMessage := err.Error()
	_, err = socket.Send("E", zmq.SNDMORE)
	logger.PanicIfError("Listener", err)
	_, err = socket.Send(errorMessage, 0)
	logger.PanicIfError("Listener", err)
}
