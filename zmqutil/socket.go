// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package zmqutil

import (
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	zmq "github.com/pebbe/zmq4"
	"time"
)

const (
	heartbeatInterval = 15 * time.Second
	heartbeatTimeout  = 60 * time.Second
	heartbeatTTL      = 120 * time.Second
)

// return a pair of connected push/pull sockets
// for shutdown signalling
func NewSignalPair(signal string) (*zmq.Socket, *zmq.Socket, error) {

	// send half of signalling channel
	push, err := zmq.NewSocket(zmq.PUSH)
	if nil != err {
		return nil, nil, err
	}
	push.SetLinger(0)
	err = push.Bind(signal)
	if nil != err {
		push.Close()
		return nil, nil, err
	}

	// receive half of signalling channel
	pull, err := zmq.NewSocket(zmq.PULL)
	if nil != err {
		push.Close()
		return nil, nil, err
	}
	pull.SetLinger(0)
	err = pull.Connect(signal)
	if nil != err {
		push.Close()
		pull.Close()
		return nil, nil, err
	}

	return push, pull, nil
}

// bind a list of addresses
// creates up to 2 sockets for separate IPv4 and IPv6 traffic
func NewBind(log *logger.L, socketType zmq.Type, zapDomain string, privateKey []byte, publicKey []byte, listen []*util.Connection) (*zmq.Socket, *zmq.Socket, error) {

	socket4 := (*zmq.Socket)(nil) // IPv4 traffic
	socket6 := (*zmq.Socket)(nil) // IPv6 traffic

	err := error(nil)

	// allocate IPv4 and IPv6 sockets
	for i, address := range listen {
		bindTo, v6 := address.CanonicalIPandPort("tcp://")
		if v6 {
			if nil == socket6 {
				socket6, err = NewServerSocket(socketType, zapDomain, privateKey, publicKey, v6)
			}
		} else {
			if nil == socket4 {
				socket4, err = NewServerSocket(socketType, zapDomain, privateKey, publicKey, v6)
			}
		}
		if nil != err {
			goto fail
		}

		if v6 {
			err = socket6.Bind(bindTo)
		} else {
			err = socket4.Bind(bindTo)
		}
		if nil != err {
			log.Errorf("cannot bind[%d]: %q  error: %v", i, bindTo, err)
			goto fail
		}
		log.Infof("bind[%d]: %q  IPv6: %v", i, bindTo, v6)

	}
	return socket4, socket6, nil

	// if an arror close any open sockets
fail:
	if nil == socket4 {
		socket4.Close()
	}
	if nil != socket6 {
		socket6.Close()
	}
	return nil, nil, err
}

// create a socket suitable for a server side connection
func NewServerSocket(socketType zmq.Type, zapDomain string, privateKey []byte, publicKey []byte, v6 bool) (*zmq.Socket, error) {

	socket, err := zmq.NewSocket(socketType)
	if nil != err {
		return nil, err
	}

	// allow any client to connect
	//zmq.AuthAllow(zapDomain, "127.0.0.1/8")
	//zmq.AuthAllow(zapDomain, "::1")
	zmq.AuthCurveAdd(zapDomain, zmq.CURVE_ALLOW_ANY)

	// domain is servers public key
	socket.SetCurveServer(1)
	//socket.SetCurvePublickey(publicKey)
	socket.SetCurveSecretkey(string(privateKey))

	socket.SetZapDomain(zapDomain)

	socket.SetIdentity(string(publicKey)) // just use public key for identity

	socket.SetIpv6(v6) // conditionally set IPv6 state

	// socket.SetSndtimeo(SEND_TIMEOUT)
	// socket.SetLinger(LINGER_TIME)
	// socket.SetRouterMandatory(0)   // discard unroutable packets
	// socket.SetRouterHandover(true) // allow quick reconnect for a given public key
	// socket.SetImmediate(false)     // queue messages sent to disconnected peer

	// heartbeat
	socket.SetHeartbeatIvl(heartbeatInterval)
	socket.SetHeartbeatTimeout(heartbeatTimeout)
	socket.SetHeartbeatTtl(heartbeatTTL)

	return socket, nil
}
