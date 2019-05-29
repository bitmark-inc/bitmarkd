// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/peer/upstream"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

// Connection - hardwired connections
// this is read from the configuration file
type Connection struct {
	PublicKey string `gluamapper:"public_key" json:"public_key"`
	Address   string `gluamapper:"address" json:"address"`
}

// Configuration - a block of configuration data
// this is read from the configuration file
type Configuration struct {
	DynamicConnections bool         `gluamapper:"dynamic_connections" json:"dynamic_connections"`
	PreferIPv6         bool         `gluamapper:"prefer_ipv6" json:"prefer_ipv6"`
	Listen             []string     `gluamapper:"listen" json:"listen"`
	Announce           []string     `gluamapper:"announce" json:"announce"`
	PrivateKey         string       `gluamapper:"private_key" json:"private_key"`
	PublicKey          string       `gluamapper:"public_key" json:"public_key"`
	Connect            []Connection `gluamapper:"connect" json:"connect,omitempty"`
}

// globals for background process
type peerData struct {
	sync.RWMutex // to allow locking

	log *logger.L // logger

	lstn listener  // for RPC responses
	conn connector // for RPC requests

	connectorClients []*upstream.Upstream

	publicKey []byte

	clientCount int
	blockHeight uint64

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData peerData

// Initialise - setup peer background processes
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
	privateKey, err := zmqutil.ReadPrivateKey(configuration.PrivateKey)
	if nil != err {
		globalData.log.Errorf("read private key file: %q  error: %s", configuration.PrivateKey, err)
		return err
	}
	publicKey, err := zmqutil.ReadPublicKey(configuration.PublicKey)
	if nil != err {
		globalData.log.Errorf("read public key file: %q  error: %s", configuration.PublicKey, err)
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

	if err := globalData.lstn.initialise(privateKey, publicKey, configuration.Listen, version); nil != err {
		return err
	}
	if err := globalData.conn.initialise(privateKey, publicKey, configuration.Connect, configuration.DynamicConnections, configuration.PreferIPv6); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	globalData.log.Info("start background…")

	processes := background.Processes{
		// &globalData.brdc,
		&globalData.lstn,
		&globalData.conn,
	}

	globalData.background = background.Start(processes, globalData.log)

	return nil
}

// configure announce so that minimum data will be present for
// connection to neighbours
func setAnnounce(configuration *Configuration, publicKey []byte) error {

	l := make([]byte, 0, 100) // ***** FIX THIS: need a better default size

process_listen:
	for i, address := range configuration.Announce {
		if "" == address {
			continue process_listen
		}
		c, err := util.NewConnection(address)
		if nil != err {
			globalData.log.Errorf("announce listen[%d]=%q  error: %s", i, address, err)
			return err
		}
		l = append(l, c.Pack()...)
	}
	if err := announce.SetPeer(publicKey, l); nil != err {
		globalData.log.Errorf("announce.SetPeer error: %s", err)
		return err
	}
	return nil
}

// Finalise - stop all background tasks
func Finalise() error {

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

// PublicKey - return public key
func PublicKey() []byte {
	return globalData.publicKey
}

// GetCounts - return connection counts:
//   incoming - total peers connectng to all listeners
//   outgoing - total outgoing connections
func GetCounts() (uint64, uint64) {
	return globalData.lstn.connections, uint64(globalData.clientCount)
}

// BlockHeight - return global block height
func BlockHeight() uint64 {
	return globalData.blockHeight
}
