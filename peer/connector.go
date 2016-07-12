// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/binary"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"time"
)

const (
	sendInterval = 30 * time.Second
)

type connector struct {
	log    *logger.L
	socket *zmq.Socket
}

// initialise the connector
func (conn *connector) initialise(configuration *Configuration) error {

	log := logger.New("connector")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	conn.log = log

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

	socket, err := zmq.NewSocket(zmq.REQ)
	if nil != err {
		return err
	}
	conn.socket = socket

	// set up as client
	socket.SetCurveServer(0)
	socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(privateKey)
	log.Tracef("client public:  %q", publicKey)
	log.Tracef("client private: %q", privateKey)

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

	// ***** FIX THIS: may not be right
	// ***** FIX THIS: maybe need seaparate socket for each connection

	for i, c := range configuration.Connect {
		address := c.Address
		serverPublicKey := c.PublicKey
		socket.SetCurveServerkey(serverPublicKey)
		log.Tracef("server public: %q", serverPublicKey)

		connectTo, err := util.CanonicalIPandPort("tcp://", address)
		if nil != err {
			log.Errorf("connector[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}

		err = socket.Connect(connectTo)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %v", i, address, err)
			socket.Close()
			return err
		}
		log.Infof("connect to: %q", address)
	}
	return nil
}

// various RPC calls to upstream connections
func (conn *connector) Run(args interface{}, shutdown <-chan struct{}) {

	log := conn.log

	log.Info("starting…")

loop:
	for {
		// wait for shutdown
		log.Info("waiting…")

		select {
		case <-shutdown:
			break loop
		case <-time.After(sendInterval):
			conn.process()
		}
	}
	conn.socket.Close()
}

var n uint64 = 0

// process the connect and return response to prooferd
func (conn *connector) process() {
	log := conn.log

	fn := "H"
	parameter := []byte{}

	n += 1
	switch n {
	case 1, 2, 3:
		log.Info("send block request")
		parameter = make([]byte, 8)
		binary.BigEndian.PutUint64(parameter, n)
		fn = "B"
	case 4:
		log.Info("info request")
		fn = "I"
	default:
		n = 0
		log.Info("send blockHeight")
	}

	_, err := conn.socket.Send(fn, zmq.SNDMORE)
	fault.PanicIfError("Connector", err)
	_, err = conn.socket.SendBytes(parameter, 0)
	fault.PanicIfError("Connector", err)

	log.Info("wait response")

	data, err := conn.socket.RecvMessageBytes(0)
	if nil != err {
		log.Errorf("receive error: %v", err)
		return
	}

	switch string(data[0]) {
	case "B":
		log.Infof("received block: %x", data[1])
	case "E":
		log.Infof("received error: %q", data[1])
	case "H":
		height := uint64(0)
		if 8 == len(data[1]) {
			height = binary.BigEndian.Uint64(data[1])
		}
		log.Infof("received block height: %x", height)
	case "I":
		var info serverInfo
		err = json.Unmarshal(data[1], &info)
		if nil != err {
			log.Errorf("JSON decode error: %v", err)
		} else {
			log.Infof("received info: %v", info)
		}
	}
}
