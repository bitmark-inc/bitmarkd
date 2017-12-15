// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/peer/upstream"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
	"sync"
)

// hardwired connections
// this is read from a libucl configuration file
type Connection struct {
	PublicKey string `libucl:"public_key" json:"public_key"`
	Address   string `libucl:"address" json:"address"`
}

// for announcements
type Announce struct {
	Broadcast []string `libucl:"broadcast" json:"broadcast"`
	Listen    []string `libucl:"listen" json:"listen"`
}

// a block of configuration data
// this is read from a libucl configuration file
type Configuration struct {
	DynamicConnections bool         `libucl:"dynamic_connections" json:"dynamic_connections"`
	Broadcast          []string     `libucl:"broadcast" json:"broadcast"`
	Listen             []string     `libucl:"listen" json:"listen"`
	Announce           Announce     `libucl:"announce" json:"announce"`
	PrivateKey         string       `libucl:"private_key" json:"private_key"`
	PublicKey          string       `libucl:"public_key" json:"public_key"`
	Subscribe          []Connection `libucl:"subscribe" json:"subscribe,omitempty"`
	Connect            []Connection `libucl:"connect" json:"connect,omitempty"`
}

// globals for background proccess
type peerData struct {
	sync.RWMutex // to allow locking

	log *logger.L // logger

	brdc broadcaster // for broadcasting blocks, transactions etc.
	lstn listener    // for RPC responses
	conn connector   // for RPC requests
	sbsc subscriber  // for subscriptions

	connectorClients  []*upstream.Upstream
	subscriberClients []*zmqutil.Client

	publicKey []byte

	blockHeight uint64

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData peerData

// initialise peer backgrouds processes
func Initialise(configuration *Configuration, version string) error {

	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("peer")
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

	globalData.publicKey = publicKey

	// set up announcer before any connections
	err = setAnnounce(configuration, publicKey)
	if nil != err {
		return err
	}

	if err := globalData.brdc.initialise(privateKey, publicKey, configuration.Broadcast); nil != err {
		return err
	}
	if err := globalData.lstn.initialise(privateKey, publicKey, configuration.Listen, version); nil != err {
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

	processes := background.Processes{
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

process_broadcast:
	for i, address := range configuration.Announce.Broadcast {
		if "" == address {
			continue process_broadcast
		}
		c, err := util.NewConnection(address)
		if nil != err {
			globalData.log.Errorf("announce broadcast[%d]=%q  error: %v", i, address, err)
			return err
		}
		b = append(b, c.Pack()...)
	}
process_listen:
	for i, address := range configuration.Announce.Listen {
		if "" == address {
			continue process_listen
		}
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

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// return public key
func PublicKey() []byte {
	return globalData.publicKey
}

// return public key
func BlockHeight() uint64 {
	return globalData.blockHeight
}
