// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package listeners

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/bitmark-inc/bitmarkd/rpc/handler"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/fault"
)

const (
	httpsLogName       = "http_rpc"
	minConnectionCount = 1
	readWriteTimeout   = 10 * time.Second
)

// HTTPSConfiguration - configuration file data for HTTPS setup
type HTTPSConfiguration struct {
	MaximumConnections uint64              `gluamapper:"maximum_connections" json:"maximum_connections"`
	Listen             []string            `gluamapper:"listen" json:"listen"`
	Certificate        string              `gluamapper:"certificate" json:"certificate"`
	PrivateKey         string              `gluamapper:"private_key" json:"private_key"`
	Allow              map[string][]string `gluamapper:"allow" json:"allow"`
}

type httpsListener struct {
	log             *logger.L
	listenIPAndPort []string
	tlsConfig       *tls.Config
	mux             *http.ServeMux
}

func (h httpsListener) Serve() error {
	for _, listen := range h.listenIPAndPort {
		h.log.Infof("starting server: %s on: %q", httpsLogName, listen)
		if '*' == listen[0] {
			// change "*:PORT" to "[::]:PORT"
			// on the assumption that this will listen on tcp4 and tcp6
			listen = "[::]" + ":" + strings.Split(listen, ":")[1]
		}

		go doServeHTTPS(listen, h.mux, h.tlsConfig)
	}

	return nil
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func doServeHTTPS(addr string, handler http.Handler, cfg *tls.Config) {
	s := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    readWriteTimeout,
		WriteTimeout:   readWriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	cfg.NextProtos = []string{"http/1.1"}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, cfg)

	_ = s.Serve(tlsListener)
}

func NewHTTPS(
	configuration *HTTPSConfiguration,
	log *logger.L,
	tlsConfig *tls.Config,
	hdlr handler.Handler,
) (Listener, error) {
	if 0 == len(configuration.Listen) {
		log.Infof("disable: %s", httpsLogName)
		return nil, nil
	}

	if configuration.MaximumConnections < minConnectionCount {
		log.Errorf("invalid %s maximum connection limit: %d", httpsLogName, configuration.MaximumConnections)
		return nil, fault.MissingParameters
	}

	h := httpsListener{
		log:             log,
		listenIPAndPort: configuration.Listen,
		tlsConfig:       tlsConfig,
	}

	// create access control and format strings to match http.Request.RemoteAddr
	local := make(map[string][]*net.IPNet)
	for path, addresses := range configuration.Allow {
		set := make([]*net.IPNet, len(addresses))
		local[path] = set
		for i, ip := range addresses {
			_, cidr, err := net.ParseCIDR(strings.Trim(ip, " "))
			if nil != err {
				return nil, err
			}
			set[i] = cidr
		}
	}

	hdlr.SetAllow(local)

	h.mux = http.NewServeMux()
	h.mux.HandleFunc("/bitmarkd/rpc", hdlr.RPC)
	h.mux.HandleFunc("/bitmarkd/details", hdlr.Details)
	h.mux.HandleFunc("/bitmarkd/connections", hdlr.Connections)
	h.mux.HandleFunc("/bitmarkd/peers", hdlr.Peers)
	h.mux.HandleFunc("/", hdlr.Root)

	return &h, nil
}
