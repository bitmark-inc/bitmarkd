// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	"sync"
)

// hardwired connections
// this is read from a libucl configuration file
type Connection struct {
	PublicKey string `libucl:"public_key"`
	Address   string `libucl:"address"`
}

// for announcements
type Announce struct {
	Broadcast []string `libucl:"broadcast"`
	Listen    []string `libucl:"listen"`
}

// a block of configuration data
// this is read from a libucl configuration file
type Configuration struct {
	DynamicConnections bool         `libucl:"dynamic_connections"`
	Broadcast          []string     `libucl:"broadcast"`
	Listen             []string     `libucl:"listen"`
	Announce           Announce     `libucl:"announce"`
	PrivateKey         string       `libucl:"private_key"`
	PublicKey          string       `libucl:"public_key"`
	Subscribe          []Connection `libucl:"subscribe"`
	Connect            []Connection `libucl:"connect"`
}

// globals for background proccess
type proofData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	brdc broadcaster // for broadcasting blocks, transactions etc.
	lstn listener    // for RPC responses
	conn connector   // for RPC requests
	sbsc subscriber  // for subscriptions

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData proofData

// initialise proofer backgrouds processes
func Initialise(configuration *Configuration) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("peer")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	// read the keys
	privateKey, err := zmqutil.ReadPrivateKeyFile(configuration.PrivateKey)
	if nil != err {
		globalData.log.Errorf("read private key file: %q  error: %v", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadPublicKeyFile(configuration.PublicKey)
	if nil != err {
		globalData.log.Errorf("read public key file: %q  error: %v", configuration.PublicKey, err)
		return err
	}
	globalData.log.Tracef("peer private key: %q", privateKey)
	globalData.log.Tracef("peer public key:  %q", publicKey)

	// set up announcer before any connections
	err = setAnnounce(configuration, publicKey)
	if nil != err {
		return err
	}

	if err := globalData.brdc.initialise(privateKey, publicKey, configuration.Broadcast); nil != err {
		return err
	}
	if err := globalData.lstn.initialise(privateKey, publicKey, configuration.Listen); nil != err {
		return err
	}
	if err := globalData.conn.initialise(privateKey, publicKey, configuration.Connect, configuration.DynamicConnections); nil != err {
		return err
	}
	if err := globalData.sbsc.initialise(privateKey, publicKey, configuration.Subscribe, configuration.DynamicConnections); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	var processes = background.Processes{
		&globalData.brdc,
		&globalData.lstn,
		&globalData.conn,
		&globalData.sbsc,
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// configure announce so that minimum data will be present for
// connection to neighbours
func setAnnounce(configuration *Configuration, publicKey []byte) error {

	b := make([]byte, 0, 100) // ***** FIX THIS: need a better default size
	l := make([]byte, 0, 100) // ***** FIX THIS: need a better default size

	for i, address := range configuration.Announce.Broadcast {
		c, err := util.NewConnection(address)
		if nil != err {
			globalData.log.Errorf("announce broadcast[%d]=%q  error: %v", i, address, err)
			return err
		}
		b = append(b, c.Pack()...)
	}
	for i, address := range configuration.Announce.Listen {
		c, err := util.NewConnection(address)
		if nil != err {
			globalData.log.Errorf("announce listen[%d]=%q  error: %v", i, address, err)
			return err
		}
		l = append(l, c.Pack()...)
	}
	if err := announce.SetPeer(publicKey, b, l); nil != err {
		globalData.log.Errorf("announce.SetPeer error: %v", err)
		return err
	}
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
	globalData.background.Stop()

	// finally...
	globalData.initialised = false

	return nil
}
