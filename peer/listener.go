// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/binary"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/version"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

const (
	listenerZapDomain = "listen"
	listenerSignal    = "inproc://bitmark-listener-signal"
)

type listener struct {
	log     *logger.L
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
func (lstn *listener) initialise(privateKey []byte, publicKey []byte, listen []string) error {

	log := logger.New("listener")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	lstn.log = log

	log.Info("initialising…")

	c, err := util.NewConnections(listen)
	if nil != err {
		log.Errorf("ip and port error: %v", err)
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
		log.Errorf("bind error: %v", err)
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
					s.Recv(0)
					break loop
				}
			}
		}
		lstn.pull.Close()
		if nil != lstn.socket4 {
			lstn.socket4.Close()
		}
		if nil != lstn.socket6 {
			lstn.socket6.Close()
		}
	}()

	// wait for shutdown
	log.Info("waiting…")
	<-shutdown
	lstn.push.SendMessage("stop")
	lstn.push.Close()
}

// process the listen and return response to client
func (lstn *listener) process(socket *zmq.Socket) {

	log := lstn.log

	log.Info("process starting…")

	data, err := socket.RecvMessage(0)
	if nil != err {
		log.Errorf("receive error: %v", err)
		return
	}

	fn := data[0]
	parameter := []byte(data[1])

	log.Infof("received message: %x", data)

	result := []byte{}

	switch fn {
	case "B": // get packed block
		if 8 == len(parameter) {
			result = storage.Pool.Blocks.Get(parameter)
			if nil == result {
				err = fault.ErrBlockNotFound
			}
		} else {
			err = fault.ErrBlockNotFound
		}

	case "I": // server information
		info := serverInfo{
			Version: version.Version,
			Chain:   mode.ChainName(),
			Normal:  mode.Is(mode.Normal),
			Height:  block.GetHeight(),
		}
		result, err = json.Marshal(info)
		fault.PanicIfError("JSON encode error: %v", err)

	case "H": // get block hash
		if 8 == len(parameter) {
			number := binary.BigEndian.Uint64(parameter)
			d, e := block.DigestForBlock(number)
			if nil == e {
				result = d[:]
			} else {
				err = e
			}
		} else {
			err = fault.ErrBlockNotFound
		}
	}

	if nil == err {
		_, err := socket.Send(fn, zmq.SNDMORE|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
		_, err = socket.SendBytes(result, 0|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
	} else {
		errorMessage := err.Error()
		_, err := socket.Send("E", zmq.SNDMORE|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
		_, err = socket.Send(errorMessage, 0|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
	}
}
