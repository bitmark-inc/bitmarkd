// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"fmt" // ***** DEBUG: IPv6 listen on fails ***** see printf below
	"github.com/bitmark-inc/bilateralrpc"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	"sync"
)

// globals for background proccess
type peerData struct {
	sync.RWMutex // to allow locking

	// maximum active clients
	//maximumConnections int // ***** FIX THIS: is it possible to do this counting *****

	// the server instance
	server *bilateralrpc.Bilateral

	// for background
	//background *background.T
	threads []*thread

	// trigger to rebroadcast
	rebroadcast bool

	// set once during initialise
	initialised bool
}

// local threads
type thread struct {
	name    string
	log     *logger.L
	stop    chan bool
	done    chan bool
	handler func(*thread)
}

// global data
var globalData peerData

// start up the peer communications system
func Initialise(addresses []string, networkName string, publicKey string, privateKey string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	// force an ititial rebroadcast
	globalData.rebroadcast = true

	globalData.threads = []*thread{
		{name: "client", handler: globalData.client},
		{name: "responder", handler: globalData.responder},
		{name: "announcer", handler: globalData.announcer},
	}

	log := logger.New("peer")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}

	for _, t := range globalData.threads {

		t.log = logger.New(t.name)
		if nil == t.log {
			return fault.ErrInvalidLoggerChannel
		}
		t.stop = make(chan bool)
		t.done = make(chan bool)
	}

	// create the server
	globalData.server = bilateralrpc.NewEncrypted(networkName, publicKey, privateKey)

	// listen on a port
	for _, address := range addresses {
		if err := globalData.server.ListenOn("tcp://" + address); nil != err {
			log.Errorf("ListenOn: tcp://%s  failed: %v\n", address, err)
			// ****** FIX THIS: currently just ignoring failed listens *****
			fmt.Printf("ListenOn: tcp://%s  failed: %v\n", address, err)
			//globalData.server.Close()
			//return err
		}
	}

	// register server objects
	// -----------------------

	// not yet....see below....
	// peer := &Peer{
	// 	log: log,
	// }

	rpcs := &RPCs{
		log: log,
	}

	cert := &Certificate{
		log: log,
	}

	asset := &Asset{
		log: log,
	}

	block := &Block{
		log: log,
	}

	transaction := &Transaction{
		log: log,
	}

	//globalData.server.Register(peer) // need some fixes first
	globalData.server.Register(rpcs)
	globalData.server.Register(cert)
	globalData.server.Register(asset)
	globalData.server.Register(block)
	globalData.server.Register(transaction)

	// all data initialised
	globalData.initialised = true

	// start backgrounds
	for _, t := range globalData.threads {
		go t.handler(t)
	}

	return nil
}

// initiate a connection
func ConnectTo(publicKey string, address string) error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	return globalData.server.ConnectTo(publicKey, "tcp://"+address)
}

// finialise - stop all background tasks
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	// stop all
	for _, t := range globalData.threads {
		close(t.stop)
	}

	// wait for all finished
	for _, t := range globalData.threads {
		<-t.done
	}

	// shutdown
	globalData.server.Close()

	// finally...
	globalData.initialised = false
	return nil
}

// count of active connections
func ConnectionCount() int {
	globalData.Lock()
	defer globalData.Unlock()
	return globalData.server.ConnectionCount()
}
