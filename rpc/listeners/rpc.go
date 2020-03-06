// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package listeners

import (
	"crypto/tls"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strings"

	"github.com/bitmark-inc/bitmarkd/counter"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

const (
	logName         = "client_rpc"
	connectionLimit = 100
	minBandwidth    = 1000000 // 1Mbps
)

type rpcListener struct {
	log             *logger.L
	listener        net.Listener
	count           *counter.Counter
	server          *rpc.Server
	maxConnections  uint64
	tlsConfig       *tls.Config
	ipType          []string
	listenIPAndPort []string
}

func (r rpcListener) Serve() error {
	var err error
	for i, listen := range r.listenIPAndPort {
		r.log.Infof("starting RPC server: %s", listen)
		r.listener, err = tls.Listen(r.ipType[i], listen, r.tlsConfig)
		if err != nil {
			r.log.Errorf("rpc server listen error: %s", err)
			return err
		}

		go doServeRPC(r.listener, r.server, r.maxConnections, r.log, r.count)
	}
	return nil
}

func doServeRPC(listen net.Listener, server *rpc.Server, maximumConnections uint64, log *logger.L, count *counter.Counter) {
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Errorf("rpc.server terminated: accept error:", err)
			break
		}
		if count.Increment() <= maximumConnections {
			go func() {
				server.ServeCodec(jsonrpc.NewServerCodec(conn))
				_ = conn.Close()
				count.Decrement()
			}()
		} else {
			count.Decrement()
			_ = conn.Close()
		}

	}
	_ = listen.Close()
	log.Error("RPC accept terminated")
}

// RPCConfiguration - configuration file data for RPC setup
type RPCConfiguration struct {
	MaximumConnections uint64   `gluamapper:"maximum_connections" json:"maximum_connections"`
	Bandwidth          float64  `gluamapper:"bandwidth" json:"bandwidth"`
	Listen             []string `gluamapper:"listen" json:"listen"`
	Certificate        string   `gluamapper:"certificate" json:"certificate"`
	PrivateKey         string   `gluamapper:"private_key" json:"private_key"`
	Announce           []string `gluamapper:"announce" json:"announce"`
}

func NewRPC(
	configuration *RPCConfiguration,
	log *logger.L,
	count *counter.Counter,
	server *rpc.Server,
	ann announce.Announce,
	tlsConfig *tls.Config,
	certificateFingerprint [32]byte,
) (Listener, error) {
	if configuration.MaximumConnections < minConnectionCount {
		log.Errorf("invalid %s maximum connection limit: %d", logName, configuration.MaximumConnections)
		return nil, fault.MissingParameters
	}
	if configuration.Bandwidth <= minBandwidth { // fail if < 1Mbps
		log.Errorf("invalid %s bandwidth: %d bps < 1Mbps", logName, configuration.Bandwidth)
		return nil, fault.MissingParameters
	}

	r := rpcListener{
		log:             log,
		maxConnections:  configuration.MaximumConnections,
		listenIPAndPort: configuration.Listen,
		server:          server,
		count:           count,
		tlsConfig:       tlsConfig,
	}

	if 0 == len(configuration.Listen) {
		log.Errorf("missing %s listen", logName)
		return nil, fault.MissingParameters
	}

	log.Infof("%s: SHA3-256 fingerprint: %x", logName, certificateFingerprint)

	// setup announce
	rpcs := make([]byte, 0, connectionLimit) // ***** FIX THIS: need a better default size

	for _, address := range configuration.Announce {
		if "" == address {
			continue
		}
		c, err := util.NewConnection(address)
		if nil != err {
			log.Errorf("invalid %s listen announce: %q  error: %s", logName, address, err)
			return nil, err
		}
		rpcs = append(rpcs, c.Pack()...)
	}

	err := ann.Set(certificateFingerprint, rpcs)
	if nil != err {
		log.Criticalf("announce.Set error: %s", err)
		return nil, err
	}

	// validate all listen addresses
	r.ipType, err = parseListenAddress(configuration.Listen, r.log)
	if nil != err {
		return nil, err
	}

	return &r, nil
}

func parseListenAddress(addrs []string, log *logger.L) ([]string, error) {
	parsed := make([]string, len(addrs))
	for i, listen := range addrs {
		if '*' == listen[0] {
			// change "*:PORT" to "[::]:PORT"
			// on the assumption that this will listen on tcp4 and tcp6
			addrs[i] = "[::]" + ":" + strings.Split(listen, ":")[1]
			listen = "::"
			parsed[i] = "tcp"
		} else if '[' == listen[0] {
			listen = strings.Split(listen[1:], "]:")[0]
			parsed[i] = "tcp6"
		} else {
			listen = strings.Split(listen, ":")[0]
			parsed[i] = "tcp4"
		}

		if ip := net.ParseIP(listen); nil == ip {
			err := fault.InvalidIpAddress
			log.Errorf("rpc server listen error: %s", err)
			return nil, err
		}
	}

	return parsed, nil
}
