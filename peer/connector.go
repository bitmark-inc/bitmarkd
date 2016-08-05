// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"time"
)

const (
	sendInterval     = 10 * time.Second
	connectorTimeout = 500 * time.Millisecond
)

type connector struct {
	log          *logger.L
	clients      []*zmqutil.Client
	dynamicStart int
}

// initialise the connector
func (conn *connector) initialise(privateKey []byte, publicKey []byte, connect []Connection, dynamicEnabled bool) error {

	log := logger.New("connector")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	conn.log = log

	log.Info("initialising…")

	// allocate all sockets
	staticCount := len(connect) // can be zero
	if 0 == staticCount && !dynamicEnabled {
		log.Error("zero static cliens and dynamic is disabled")
		return fault.ErrNoConnectionsAvailable
	}
	conn.clients = make([]*zmqutil.Client, staticCount+offsetCount)
	conn.dynamicStart = staticCount // index of first dynamic socket

	// error code for goto fail
	errX := error(nil)

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

		client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}

		conn.clients[i] = client

		err = client.Connect(address, serverPublicKey)
		if nil != err {
			log.Errorf("connect[%d]=%q  error: %v", i, address, err)
			errX = err
			goto fail
		}
		log.Infof("public key: %x  at: %q", serverPublicKey, c.Address)
	}

	// just create sockets for dynamic clients
	for i := conn.dynamicStart; i < len(conn.clients); i += 1 {
		client, err := zmqutil.NewClient(zmq.REQ, privateKey, publicKey, connectorTimeout)
		if nil != err {
			log.Errorf("client[%d]  error: %v", i, err)
			errX = err
			goto fail
		}

		conn.clients[i] = client
	}
	return nil

	// error handling
fail:
	zmqutil.CloseClients(conn.clients)
	return errX
}

// various RPC calls to upstream connections
func (conn *connector) Run(args interface{}, shutdown <-chan struct{}) {

	log := conn.log

	log.Info("starting…")

	queue := messagebus.Bus.Connector.Chan()

loop:
	for {
		// wait for shutdown
		log.Info("waiting…")

		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			conn.log.Infof("received: %s  public key: %x  connect: %x", item.Command, item.Parameters[0], item.Parameters[1])
			connectTo(conn.log, conn.clients, conn.dynamicStart, item.Command, item.Parameters[0], item.Parameters[1])

		case <-time.After(sendInterval):
			for i, c := range conn.clients {
				if c.IsConnected() {
					conn.log.Infof("transaction to client: %d", i)
					conn.process(c)
				}
			}
		}
	}
	zmqutil.CloseClients(conn.clients)
}

var n int64 = -3

// process the connect and return response
func (conn *connector) process(client *zmqutil.Client) {
	log := conn.log

	fn := "I"
	parameter := []byte{}

	err := error(nil)
	sent := false

	switch n {
	case 1, 2, 3, 4, 5:
		m := uint64(n)
		log.Infof("send block request: %d", m)
		parameter = make([]byte, 8)
		binary.BigEndian.PutUint64(parameter, m)
		fn = "B"
	case 6, 7, 8, 9, 10, 11:
		m := uint64(n - 5) // make as 1...
		log.Infof("send block hash request: %d", m)
		parameter = make([]byte, 8)
		binary.BigEndian.PutUint64(parameter, m)
		fn = "H"
	case -2, 0, 12:
		log.Info("send registration request")
		err = announce.SendRegistration(client, "R")
		sent = true

	case -1:
		log.Info("info request")
		fn = "I"

	default:
		log.Info("info request")
		fn = "I"
		n = 0
	}
	n += 1

	// if no message sent the use default send process
	if !sent {
		err = client.Send(fn, parameter)
	}
	if nil != err {
		log.Errorf("send error: %v", err)
		err := client.Reconnect()
		if nil != err {
			log.Errorf("reconnect error: %v", err)
		}
		return
	}

	log.Info("wait response")

	data, err := client.Receive(0)
	if nil != err {
		log.Errorf("receive error: %v", err)
		err := client.Reconnect()
		if nil != err {
			log.Errorf("reconnect error: %v", err)
		}
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

	case "R":
		announce.AddPeer(data[1], data[2], data[3])                      // publicKey, broadcasts, listeners
		messagebus.Bus.Broadcast.Send("peer", data[1], data[2], data[3]) // publicKey, broadcasts, listeners

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
