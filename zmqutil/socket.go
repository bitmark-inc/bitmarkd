// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
)

// point at which to disconnect large message senders
// current estimate of a block maximum is 2 MB
const (
	maximumPacketSize = 5000000 // 5 MB
)

// ***** FIX THIS: enabling this causes complete failure
// ***** FIX THIS: socket disconnects, perhaps after IVL value
// const (
// 	heartbeatInterval = 15 * time.Second
// 	heartbeatTimeout  = 60 * time.Second
// 	heartbeatTTL      = 60 * time.Second
// )

// NewSignalPair -return a pair of connected PAIR sockets
// for shutdown signalling
func NewSignalPair(signal string) (receiver *zmq.Socket, sender *zmq.Socket, err error) {

	// PAIR server, half of signalling channel
	receiver, err = zmq.NewSocket(zmq.PAIR)
	if err != nil {
		return nil, nil, err
	}
	receiver.SetLinger(0)
	err = receiver.Bind(signal)
	if err != nil {
		receiver.Close()
		return nil, nil, err
	}

	// PAIR Client, half of signalling channel
	sender, err = zmq.NewSocket(zmq.PAIR)
	if err != nil {
		receiver.Close()
		sender.Close()
		return nil, nil, err
	}
	sender.SetLinger(0)
	err = sender.Connect(signal)
	if err != nil {
		receiver.Close()
		sender.Close()
		return nil, nil, err
	}

	return receiver, sender, nil
}

// NewBind - bind socket to a list of addresses
// creates up to 2 sockets for separate IPv4 and IPv6 traffic
func NewBind(log *logger.L, socketType zmq.Type, zapDomain string, privateKey []byte, publicKey []byte, listen []*util.Connection) (*zmq.Socket, *zmq.Socket, error) {

	socket4 := (*zmq.Socket)(nil) // IPv4 traffic
	socket6 := (*zmq.Socket)(nil) // IPv6 traffic

	err := error(nil)

	// allocate IPv4 and IPv6 sockets
	for i, address := range listen {
		bindTo, v6 := address.CanonicalIPandPort("tcp://")
		if v6 {
			if socket6 == nil {
				socket6, err = NewServerSocket(socketType, zapDomain, privateKey, publicKey, v6)
			}
		} else {
			if socket4 == nil {
				socket4, err = NewServerSocket(socketType, zapDomain, privateKey, publicKey, v6)
			}
		}
		if err != nil {
			goto fail
		}

		if v6 {
			err = socket6.Bind(bindTo)
		} else {
			err = socket4.Bind(bindTo)
		}
		if err != nil {
			log.Errorf("cannot bind[%d]: %q  error: %s", i, bindTo, err)
			goto fail
		}
		log.Infof("bind[%d]: %q  IPv6: %v", i, bindTo, v6)

	}
	return socket4, socket6, nil

	// if an error close any open sockets
fail:
	if socket4 != nil {
		socket4.Close()
	}
	if socket6 != nil {
		socket6.Close()
	}
	log.Errorf("socket error: %s", err)
	return nil, nil, err
}

// NewServerSocket - create a socket suitable for a server side connection
func NewServerSocket(socketType zmq.Type, zapDomain string, privateKey []byte, publicKey []byte, v6 bool) (*zmq.Socket, error) {

	socket, err := zmq.NewSocket(socketType)
	if err != nil {
		return nil, err
	}

	// all errors after here must goto failure to ensure proper
	// cleanup

	// allow any client to connect
	//zmq.AuthAllow(zapDomain, "127.0.0.1/8")
	//zmq.AuthAllow(zapDomain, "::1")
	zmq.AuthCurveAdd(zapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(string(privateKey))

	err = socket.SetZapDomain(zapDomain)
	if err != nil {
		goto failure
	}

	// use public key for identity
	socket.SetIdentity(string(publicKey))

	// conditionally set IPv6 state
	err = socket.SetIpv6(v6)
	if err != nil {
		goto failure
	}

	// only queue message to connected peers
	socket.SetImmediate(true)
	socket.SetLinger(100 * time.Millisecond)

	err = socket.SetSndtimeo(120 * time.Second)
	if err != nil {
		goto failure
	}
	err = socket.SetRcvtimeo(120 * time.Second)
	if err != nil {
		goto failure
	}

	err = socket.SetTcpKeepalive(1)
	if err != nil {
		goto failure
	}
	err = socket.SetTcpKeepaliveCnt(5)
	if err != nil {
		goto failure
	}
	err = socket.SetTcpKeepaliveIdle(60)
	if err != nil {
		goto failure
	}
	err = socket.SetTcpKeepaliveIntvl(60)
	if err != nil {
		goto failure
	}

	// ***** FIX THIS: enabling this causes complete failure
	// ***** FIX THIS: socket disconnects, perhaps after IVL value
	// heartbeat
	// err = socket.SetHeartbeatIvl(heartbeatInterval)
	// if err != nil {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTimeout(heartbeatTimeout)
	// if err != nil {
	// 	goto failure
	// }
	// err = socket.SetHeartbeatTtl(heartbeatTTL)
	// if err != nil {
	// 	goto failure
	// }

	err = socket.SetMaxmsgsize(maximumPacketSize)
	if err != nil {
		goto failure
	}

	return socket, nil

failure:
	return nil, err
}
