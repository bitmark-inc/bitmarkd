// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.e
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/rpc/listeners"

	"github.com/bitmark-inc/bitmarkd/rpc/certificate"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc/server"
	"github.com/bitmark-inc/logger"
)

const (
	tlsName = "client_rpc"
)

// HTTPSConfiguration - configuration file data for HTTPS setup
type HTTPSConfiguration struct {
	MaximumConnections uint64              `gluamapper:"maximum_connections" json:"maximum_connections"`
	Listen             []string            `gluamapper:"listen" json:"listen"`
	Certificate        string              `gluamapper:"certificate" json:"certificate"`
	PrivateKey         string              `gluamapper:"private_key" json:"private_key"`
	Allow              map[string][]string `gluamapper:"allow" json:"allow"`
}

// globals
type rpcData struct {
	sync.RWMutex // to allow locking

	log *logger.L // logger

	// set once during initialise
	initialised bool
}

// global data
var globalData rpcData

// Initialise - setup peer background processes
func Initialise(rpcConfiguration *listeners.RPCConfiguration, httpsConfiguration *HTTPSConfiguration, version string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to Start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	log := logger.New("rpc")
	globalData.log = log
	log.Info("starting…")

	tlsConfig, certificateFingerprint, err := certificate.Get(globalData.log, tlsName, rpcConfiguration.Certificate, rpcConfiguration.PrivateKey)

	// servers
	rpcListener, err := listeners.NewRPCListener(
		rpcConfiguration,
		log,
		&connectionCountRPC,
		server.Create(log, version, &connectionCountRPC),
		announce.Get(),
		tlsConfig,
		certificateFingerprint,
	)
	if nil != err {
		return err
	}
	err = rpcListener.Serve()
	if nil != err {
		return err
	}

	err = initialiseHTTPS(httpsConfiguration, version)
	if nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	return nil
}

// Finalise - stop all background tasks
func Finalise() error {

	if !globalData.initialised {
		return fault.NotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// Start server with Test instance as a service
func initialiseHTTPS(configuration *HTTPSConfiguration, version string) error {

	name := "http_rpc"
	log := globalData.log

	if 0 == len(configuration.Listen) {
		log.Infof("disable: %s", name)
		return nil
	}

	if configuration.MaximumConnections < 1 {
		log.Errorf("invalid %s maximum connection limit: %d", name, configuration.MaximumConnections)
		return fault.MissingParameters
	}

	tlsConfiguration, fingerprint, err := certificate.Get(globalData.log, name, configuration.Certificate, configuration.PrivateKey)
	if nil != err {
		return err
	}

	log.Infof("%s: SHA3-256 fingerprint: %x", name, fingerprint)

	// create access control and format strings to match http.Request.RemoteAddr
	local := make(map[string][]*net.IPNet)
	for path, addresses := range configuration.Allow {
		set := make([]*net.IPNet, len(addresses))
		local[path] = set
		for i, ip := range addresses {
			_, cidr, err := net.ParseCIDR(strings.Trim(ip, " "))
			if nil != err {
				return err
			}
			set[i] = cidr
		}
	}

	s := server.Create(log, version, &connectionCountRPC)
	handler := &httpHandler{
		log:                log,
		server:             s,
		version:            version,
		start:              time.Now(),
		allow:              local,
		maximumConnections: configuration.MaximumConnections,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/bitmarkd/rpc", handler.rpc)
	mux.HandleFunc("/bitmarkd/details", handler.details)
	mux.HandleFunc("/bitmarkd/connections", handler.connections)
	mux.HandleFunc("/bitmarkd/peers", handler.peers)
	mux.HandleFunc("/", handler.root)

	for _, listen := range configuration.Listen {
		log.Infof("starting server: %s on: %q", name, listen)
		if '*' == listen[0] {
			// change "*:PORT" to "[::]:PORT"
			// on the assumption that this will listen on tcp4 and tcp6
			listen = "[::]" + ":" + strings.Split(listen, ":")[1]
		}
		go ListenAndServeTLSKeyPair(listen, mux, tlsConfiguration)
	}

	return nil
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// ListenAndServeTLSKeyPair - Start a HTTPS server using in-memory TLS KeyPair
func ListenAndServeTLSKeyPair(addr string, handler http.Handler, cfg *tls.Config) error {
	s := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	cfg.NextProtos = []string{"http/1.1"}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, cfg)

	return s.Serve(tlsListener)
}
