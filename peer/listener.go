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
	log    *logger.L
	push   *zmq.Socket
	pull   *zmq.Socket
	socket *zmq.Socket
}

// type to hold server info
type serverInfo struct {
	Version string `json:"version"`
	Chain   string `json:"chain"`
	Normal  bool   `json:"normal"`
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
	err = push.Bind(listenerSignal)
	if nil != err {
		return err
	}

	listen.push = push

	// receive half of signalling channel
	pull, err := zmq.NewSocket(zmq.PULL)
	if nil != err {
		return err
	}
	pull.SetLinger(0)
	err = pull.Connect(listenerSignal)
	if nil != err {
		return err
	}

	listen.pull = pull

	socket, err := zmq.NewSocket(zmq.REP)
	if nil != err {
		return err
	}
	listen.socket = socket

	// ***** FIX THIS ****
	// this allows any client to connect
	zmq.AuthAllow(listenerZapDomain, "127.0.0.1/8")
	zmq.AuthCurveAdd(listenerZapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(privateKey)
	log.Tracef("server public:  %q", publicKey)
	log.Tracef("server private: %q", privateKey)

	socket.SetZapDomain(listenerZapDomain)

	socket.SetIdentity(publicKey) // just use public key for identity

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
	for i, address := range configuration.Listen {
		bindTo, err := util.CanonicalIPandPort("tcp://", address)
		if nil != err {
			log.Errorf("listener[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}

		err = socket.Bind(bindTo)
		if nil != err {
			log.Errorf("submit[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}
		log.Infof("listen on: %q", address)
	}
	return nil
}

// wait for incoming requests, process them and reply
func (listen *listener) Run(args interface{}, shutdown <-chan struct{}) {

	log := listen.log

	log.Info("starting…")

	go func() {
		poller := zmq.NewPoller()
		poller.Add(listen.socket, zmq.POLLIN)
		poller.Add(listen.pull, zmq.POLLIN)
	loop:
		for {
			sockets, _ := poller.Poll(-1)
			for _, socket := range sockets {
				switch s := socket.Socket; s {
				case listen.socket:
					listen.process()
				case listen.pull:
					s.Recv(0)
					break loop
				}
			}
		}
		listen.pull.Close()
		listen.socket.Close()
	}()

	// wait for shutdown
	log.Info("waiting…")
	<-shutdown
	listen.push.SendMessage("stop")
	listen.push.Close()
}

// process the listen and return response to client
func (listen *listener) process() {

	log := listen.log

	log.Info("process starting…")

	data, err := listen.socket.RecvMessage(0)
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

	case "I":
		info := serverInfo{
			Version: version.Version,
			Chain:   mode.ChainName(),
			Normal:  mode.Is(mode.Normal),
		}
		result, err = json.Marshal(info)
		fault.PanicIfError("JSON encode error: %v", err)

	case "H": // get block height
		number := block.GetHeight()
		result = make([]byte, 8)
		binary.BigEndian.PutUint64(result, number)
	}

	if nil == err {
		_, err := listen.socket.Send(fn, zmq.SNDMORE|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
		_, err = listen.socket.SendBytes(result, 0|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
	} else {
		errorMessage := err.Error()
		_, err := listen.socket.Send("E", zmq.SNDMORE|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
		_, err = listen.socket.Send(errorMessage, 0|zmq.DONTWAIT)
		fault.PanicIfError("Listener", err)
	}
}
