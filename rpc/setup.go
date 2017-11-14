// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"crypto/tls"
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/listener"
	"github.com/bitmark-inc/logger"
	"golang.org/x/crypto/sha3"
	"net"
	"net/http"
	"net/rpc"
	"strings"
	"sync"
	"time"
)

type RPCConfiguration struct {
	MaximumConnections int      `libucl:"maximum_connections" json:"maximum_connections"`
	Listen             []string `libucl:"listen" json:"listen"`
	Certificate        string   `libucl:"certificate" json:"certificate"`
	PrivateKey         string   `libucl:"private_key" json:"private_key"`
	Announce           []string `libucl:"announce" json:"announce"`
}

type HTTPSConfiguration struct {
	MaximumConnections int      `libucl:"maximum_connections" json:"maximum_connections"`
	Listen             []string `libucl:"listen" json:"listen"`
	Certificate        string   `libucl:"certificate" json:"certificate"`
	PrivateKey         string   `libucl:"private_key" json:"private_key"`
	LocalAllow         []string `libucl:"local_allow" json:"local_allow"`
}

// globals
type rpcData struct {
	sync.RWMutex // to allow locking

	log *logger.L // logger

	listener *listener.MultiListener

	httpServer *httpHandler

	// set once during initialise
	initialised bool
}

// global data
var globalData rpcData

// initialise peer backgrouds processes
func Initialise(rpcConfiguration *RPCConfiguration, httpsConfiguration *HTTPSConfiguration, version string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
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

// finialise - stop all background tasks
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// stop background
	globalData.listener.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

func initialiseRPC(configuration *RPCConfiguration, version string) error {
	name := "client_rpc"
	log := globalData.log

	if configuration.MaximumConnections <= 0 {
		log.Errorf("invalid %s maximum connection limit: %d", name, configuration.MaximumConnections)
		return fault.ErrMissingParameters
	}

	if 0 == len(configuration.Listen) {
		log.Errorf("missing %s listen", name)
		return fault.ErrMissingParameters
	}

	// create limiter
	limiter := listener.NewLimiter(configuration.MaximumConnections)

	tlsConfiguration, fingerprint, err := getCertificate(log, name, configuration.Certificate, configuration.PrivateKey)
	if nil != err {
		return err
	}

	log.Infof("%s: SHA3-256 fingerprint: %x", name, fingerprint)

	log.Infof("multi listener for: %s", name)
	ml, err := listener.NewMultiListener(name, configuration.Listen, tlsConfiguration, limiter, Callback)
	if nil != err {
		log.Errorf("invalid %s listen addresses", name)
		return err
	}
	globalData.listener = ml

	// setup announce
	rpcs := make([]byte, 0, 100) // ***** FIX THIS: need a better default size
process_rpcs:
	for _, address := range configuration.Announce {
		if "" == address {
			continue process_rpcs
		}
		c, err := util.NewConnection(address)
		if nil != err {
			log.Errorf("invalid %s listen announce: %q  error: %v", name, address, err)
			return err
		}
		rpcs = append(rpcs, c.Pack()...)
	}
	err = announce.SetRPC(fingerprint, rpcs)
	if nil != err {
		log.Criticalf("announce.SetRPC error: %s", err)
		return err
	}

	server, _ := createRPCServer(log, version)
	argument := &serverArgument{
		Log:    log,
		Server: server,
	}

	log.Infof("starting server: %s  with: %v", name, argument)
	globalData.listener.Start(argument)

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

	if configuration.MaximumConnections <= 0 {
		log.Errorf("invalid %s maximum connection limit: %d", name, configuration.MaximumConnections)
		return fault.ErrMissingParameters
	}

	tlsConfiguration, fingerprint, err := getCertificate(globalData.log, name, configuration.Certificate, configuration.PrivateKey)
	if nil != err {
		return err
	}

	log.Infof("%s: SHA3-256 fingerprint: %x", name, fingerprint)

	// create access control and format strings to match http.Request.RemoteAddr
	local := make(map[string]struct{})
local_loop:
	for _, la := range configuration.LocalAllow {
		ip := net.ParseIP(strings.Trim(la, " "))
		if nil == ip {
			continue local_loop
		}
		if nil != ip.To4() {
			local[ip.String()] = struct{}{}
		} else {

			local["["+ip.String()+"]"] = struct{}{}
		}
	}

	server, node := createRPCServer(log, version)
	handler := &httpHandler{
		Log:        log,
		Server:     server,
		Node:       node,
		LocalAllow: local,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/bitmarkd/rpc", handler.rpc)
	mux.HandleFunc("/bitmarkd/info", handler.info)
	mux.HandleFunc("/bitmarkd/info/connectors", handler.connectors)
	mux.HandleFunc("/bitmarkd/info/subscribers", handler.subscribers)
	mux.HandleFunc("/bitmarkd/local/peers", handler.peers)
	mux.HandleFunc("/", handler.root)

	for _, listen := range configuration.Listen {
		log.Infof("starting server: %s on: %q", name, listen)
		s := &http.Server{
			Addr:           listen,
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			TLSConfig:      tlsConfiguration,
		}

		go func() {
			err := s.ListenAndServeTLS(configuration.Certificate, configuration.PrivateKey)
			log.Errorf("server: %s on: %q  error: %s", name, listen, err)
		}()
	}

	return nil
}

func createRPCServer(log *logger.L, version string) (*rpc.Server, *Node) {

	start := time.Now().UTC()

	assets := &Assets{
		log: log,
	}

	bitmark := &Bitmark{
		log: log,
	}

	bitmarks := &Bitmarks{
		log: log,
	}

	owner := &Owner{
		log: log,
	}

	node := &Node{
		log:     log,
		start:   start,
		version: version,
	}

	transaction := &Transaction{
		log:   log,
		start: start,
	}

	server := rpc.NewServer()

	server.Register(assets)
	server.Register(bitmark)
	server.Register(bitmarks)
	server.Register(owner)
	server.Register(node)
	server.Register(transaction)

	return server, node
}

// Verify that a set of listener parameters are valid
// and return the certificate
func getCertificate(log *logger.L, name string, certificateFileName string, keyFileName string) (*tls.Config, [32]byte, error) {

	var fingerprint [32]byte

	if !util.EnsureFileExists(certificateFileName) {
		log.Errorf("certificate: %q does not exist", certificateFileName)
		return nil, fingerprint, fault.ErrCertificateFileNotFound
	}

	if !util.EnsureFileExists(keyFileName) {
		log.Errorf("private key: %q does not exist", keyFileName)
		return nil, fingerprint, fault.ErrKeyFileNotFound
	}

	// set up TLS
	keyPair, err := tls.LoadX509KeyPair(certificateFileName, keyFileName)
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

// compute the fingerprint of a certificate
//
// FreeBSD: openssl x509 -outform DER -in bitmarkd-local-rpc.crt | sha3sum -a 256
func CertificateFingerprint(certificate []byte) [32]byte {
	return sha3.Sum256(certificate)
}
