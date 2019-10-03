// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/sha3"
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// RPCConfiguration - configuration file data for RPC setup
type RPCConfiguration struct {
	MaximumConnections uint64   `gluamapper:"maximum_connections" json:"maximum_connections"`
	Bandwidth          float64  `gluamapper:"bandwidth" json:"bandwidth"`
	Listen             []string `gluamapper:"listen" json:"listen"`
	Certificate        string   `gluamapper:"certificate" json:"certificate"`
	PrivateKey         string   `gluamapper:"private_key" json:"private_key"`
	Announce           []string `gluamapper:"announce" json:"announce"`
}

// HTTPSConfiguration - configuration file data for HTTPS setup
type HTTPSConfiguration struct {
	MaximumConnections uint64              `gluamapper:"maximum_connections" json:"maximum_connections"`
	Listen             []string            `gluamapper:"listen" json:"listen"`
	Certificate        string              `gluamapper:"certificate" json:"certificate"`
	PrivateKey         string              `gluamapper:"private_key" json:"private_key"`
	Allow              map[string][]string `gluamapper:"allow" json:"allow"`
}

// rate limiting (requests per second)
// burst limit   (total items in one request)
const (
	rateLimitAssets = 200
	rateBurstAssets = 100

	rateLimitBitmark = 200
	rateBurstBitmark = 100

	rateLimitBitmarks = 200
	rateBurstBitmarks = reservoir.MaximumIssues

	rateLimitOwner = 200
	rateBurstOwner = 100

	rateLimitNode = 200
	rateBurstNode = 100

	rateLimitTransaction = 200
	rateBurstTransaction = 100

	rateLimitBlockOwner = 200
	rateBurstBlockOwner = 100

	rateLimitShare = 200
	rateBurstShare = 100
)

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
func Initialise(rpcConfiguration *RPCConfiguration, httpsConfiguration *HTTPSConfiguration, version string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	log := logger.New("rpc")
	globalData.log = log
	log.Info("starting…")

	// servers
	err := initialiseRPC(rpcConfiguration, version)
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

	// stop background
	//globalData.listener.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

func initialiseRPC(configuration *RPCConfiguration, version string) error {
	name := "client_rpc"
	log := globalData.log

	if configuration.MaximumConnections < 1 {
		log.Errorf("invalid %s maximum connection limit: %d", name, configuration.MaximumConnections)
		return fault.MissingParameters
	}
	if configuration.Bandwidth <= 1000000 { // fail if < 1Mbps
		log.Errorf("invalid %s bandwidth: %d bps < 1Mbps", name, configuration.Bandwidth)
		return fault.MissingParameters
	}

	if 0 == len(configuration.Listen) {
		log.Errorf("missing %s listen", name)
		return fault.MissingParameters
	}

	// create limiter
	//	limiter := listener.NewLimiter(configuration.MaximumConnections)

	tlsConfiguration, fingerprint, err := getCertificate(log, name, configuration.Certificate, configuration.PrivateKey)
	if nil != err {
		return err
	}

	log.Infof("%s: SHA3-256 fingerprint: %x", name, fingerprint)

	// setup announce
	rpcs := make([]byte, 0, 100) // ***** FIX THIS: need a better default size
process_rpcs:
	for _, address := range configuration.Announce {
		if "" == address {
			continue process_rpcs
		}
		c, err := util.NewConnection(address)
		if nil != err {
			log.Errorf("invalid %s listen announce: %q  error: %s", name, address, err)
			return err
		}
		rpcs = append(rpcs, c.Pack()...)
	}
	err = announce.SetRPC(fingerprint, rpcs)
	if nil != err {
		log.Criticalf("announce.SetRPC error: %s", err)
		return err
	}

	server := createRPCServer(log, version)

	for _, listen := range configuration.Listen {
		log.Infof("starting RPC server: %s", listen)
		l, err := tls.Listen("tcp", listen, tlsConfiguration)
		if err != nil {
			log.Errorf("rpc server listen error: %s", err)
			return err
		}

		go listenAndServeRPC(l, server, configuration.MaximumConnections, log)
	}

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

	tlsConfiguration, fingerprint, err := getCertificate(globalData.log, name, configuration.Certificate, configuration.PrivateKey)
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

	server := createRPCServer(log, version)
	handler := &httpHandler{
		log:                log,
		server:             server,
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

		go ListenAndServeTLSKeyPair(listen, mux, tlsConfiguration)
	}

	return nil
}

func createRPCServer(log *logger.L, version string) *rpc.Server {

	start := time.Now().UTC()

	assets := &Assets{
		log:     log,
		limiter: rate.NewLimiter(rateLimitAssets, rateBurstAssets),
	}

	bitmark := &Bitmark{
		log:     log,
		limiter: rate.NewLimiter(rateLimitBitmark, rateBurstBitmark),
	}

	bitmarks := &Bitmarks{
		log:     log,
		limiter: rate.NewLimiter(rateLimitBitmarks, rateBurstBitmarks),
	}

	owner := &Owner{
		log:     log,
		limiter: rate.NewLimiter(rateLimitOwner, rateBurstOwner),
	}

	node := &Node{
		log:     log,
		limiter: rate.NewLimiter(rateLimitNode, rateBurstNode),
		start:   start,
		version: version,
	}

	transaction := &Transaction{
		log:     log,
		limiter: rate.NewLimiter(rateLimitTransaction, rateBurstTransaction),
		start:   start,
	}

	blockOwner := &BlockOwner{
		log:     log,
		limiter: rate.NewLimiter(rateLimitBlockOwner, rateBurstBlockOwner),
	}

	share := &Share{
		log:     log,
		limiter: rate.NewLimiter(rateLimitShare, rateBurstShare),
	}

	server := rpc.NewServer()

	server.Register(assets)
	server.Register(bitmark)
	server.Register(bitmarks)
	server.Register(owner)
	server.Register(node)
	server.Register(transaction)
	server.Register(blockOwner)
	server.Register(share)

	return server
}

// Verify that a set of listener parameters are valid
// and return the certificate
func getCertificate(log *logger.L, name, certificate, key string) (*tls.Config, [32]byte, error) {
	var fingerprint [32]byte

	keyPair, err := tls.X509KeyPair([]byte(certificate), []byte(key))
	if err != nil {
		log.Errorf("%s failed to load keypair: %v", name, err)
		return nil, fingerprint, err
	}

	tlsConfiguration := &tls.Config{
		Certificates: []tls.Certificate{
			keyPair,
		},
	}

	fingerprint = CertificateFingerprint(keyPair.Certificate[0])

	return tlsConfiguration, fingerprint, nil
}

// CertificateFingerprint - compute the fingerprint of a certificate
//
// FreeBSD: openssl x509 -outform DER -in bitmarkd-local-rpc.crt | sha3sum -a 256
func CertificateFingerprint(certificate []byte) [32]byte {
	return sha3.Sum256(certificate)
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

// ListenAndServeTLSKeyPair - start a HTTPS server using in-memory TLS KeyPair
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
