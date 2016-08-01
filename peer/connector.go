// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"time"
)

const (
	sendInterval   = 10 * time.Second
	reqTimeout     = 100 * time.Millisecond
	dynamicClients = 9 // maximum dynamic clients to connect to
)

type connector struct {
	log     *logger.L
	static  bool
	clients []*zmqutil.Client
}

// initialise the connector
func (conn *connector) initialise(privateKey []byte, publicKey []byte, connect []Connection) error {

	log := logger.New("connector")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	conn.log = log

	log.Info("initialising…")

	errX := error(nil)

	staticClients := len(connect)
	if 0 == staticClients {
		conn.clients = make([]*zmqutil.Client, dynamicClients)
		conn.static = false
		for i := 0; i < dynamicClients; i += 1 {
			client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, reqTimeout)
			if nil != err {
				log.Errorf("client[%d]  error: %v", i, err)
				errX = err
				goto fail
			}

			conn.clients[i] = client
		}
		return nil
	} else {
		conn.clients = make([]*zmqutil.Client, staticClients)
		conn.static = true
	}

	// initially connect all static sockets
	for i, c := range connect {
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

		client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, reqTimeout)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}

		conn.clients[i] = client

		err = client.Connect(address, serverPublicKey)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %v", i, c.Address, err)
			errX = err
			goto fail
		}
		log.Infof("public key: %x  at: %q", serverPublicKey, c.Address)
	}

	return nil

	// error handling
fail:
	zmqutil.CloseClients(conn.clients)
	return errX
}

// request a new connection
//
// the oldest connection will be disconnected and replaced by this new
// connection
func (conn *connector) Connect(to string) {
}

// various RPC calls to upstream connections
func (conn *connector) Run(args interface{}, shutdown <-chan struct{}) {

	log := conn.log

	log.Info("starting…")

	v4 := true
	n := 0
loop:
	for {
		// wait for shutdown
		log.Info("waiting…")

		select {
		case <-shutdown:
			break loop
		case <-time.After(sendInterval):
			if v4 {
				conn.process(conn.clients[0])
			} else {
				conn.process(conn.clients[1])
			}

			n += 1
			if n >= 4 {
				n = 0
				v4 = !v4
			}
		}
	}
	zmqutil.CloseClients(conn.clients)
}

var n uint64 = 0

// process the connect and return response
func (conn *connector) process(client *zmqutil.Client) {
	log := conn.log

	fn := "I"
	parameter := []byte{}

	n += 1
	switch n {
	case 1, 2, 3, 4, 5:
		log.Infof("send block request: %d", n)
		parameter = make([]byte, 8)
		binary.BigEndian.PutUint64(parameter, n)
		fn = "B"
	case 6, 7, 8, 9, 10, 11:
		m := n - 5 // make as 1...
		log.Infof("send block hash request: %d", m)
		parameter = make([]byte, 8)
		binary.BigEndian.PutUint64(parameter, m)
		fn = "H"
	default:
		n = 0
		log.Info("info request")
		fn = "I"
	}

	err := client.Send(fn, parameter)
	//fault.PanicIfError("Connector", err)
	if nil != err {
		log.Errorf("send error: %v", err)
		client.Reconnect()
		return
	}

	log.Info("wait response")

	data, err := client.Receive(0)
	if nil != err {
		log.Errorf("receive error: %v", err)
		client.Reconnect()
		return
	}

	switch string(data[0]) {
	case "B":
		log.Infof("received block: %x", data[1])
	case "E":
		log.Errorf("received error: %q", data[1])
	case "H":
		d := blockdigest.Digest{}
		if blockdigest.Length == len(data[1]) {
			err := blockdigest.DigestFromBytes(&d, data[1])
			if nil != err {
				log.Errorf("digest decode error: %v", err)
			} else {
				log.Infof("received block digest: %v", d)
			}
		}

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
