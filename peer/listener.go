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
func (listen *listener) initialise(configuration *Configuration) error {

	log := logger.New("listener")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	listen.log = log

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
	log.Tracef("server public:  %x", publicKey)
	log.Tracef("server private: %x", privateKey)

	// signalling channel
	listen.push, listen.pull, err = zmqutil.NewSignalPair(listenerSignal)
	if nil != err {
		return err
	}

	// allocate IPv4 and IPv6 sockets
	listen.socket4, listen.socket6, err = zmqutil.NewBind(log, zmq.REP, listenerZapDomain, privateKey, publicKey, configuration.Listen)
	if nil != err {
		log.Errorf("bind error: %v", err)
		return err
	}

	return nil
}

// wait for incoming requests, process them and reply
func (listen *listener) Run(args interface{}, shutdown <-chan struct{}) {

	log := listen.log

	log.Info("starting…")

	go func() {
		poller := zmq.NewPoller()
		if nil != listen.socket4 {
			poller.Add(listen.socket4, zmq.POLLIN)
		}
		if nil != listen.socket6 {
			poller.Add(listen.socket6, zmq.POLLIN)
		}
		poller.Add(listen.pull, zmq.POLLIN)
	loop:
		for {
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case listen.socket4:
					listen.process(listen.socket4)
				case listen.socket6:
					listen.process(listen.socket6)
				case listen.pull:
					s.Recv(0)
					break loop
				}
			}
		}
		listen.pull.Close()
		if nil != listen.socket4 {
			listen.socket4.Close()
		}
		if nil != listen.socket6 {
			listen.socket6.Close()
		}
	}()

	// wait for shutdown
	log.Info("waiting…")
	<-shutdown
	listen.push.SendMessage("stop")
	listen.push.Close()
}

// process the listen and return response to client
func (listen *listener) process(socket *zmq.Socket) {

	log := listen.log

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
