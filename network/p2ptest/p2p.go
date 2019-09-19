package main

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	p2pcore "github.com/libp2p/go-libp2p-core"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
)

// global data
var globalData P2PNode

// const
const (
	version       = "v0.0.1"
	domainLocal   = "nodes.rachael.bitmark"
	domainBitamrk = "nodes.test.bitmark.com"
	domainTest    = "nodes.test.bitmark.com"
)

// StaticConnection - hardwired connections
// this is read from the configuration file
type StaticConnection struct {
	PublicKey string `gluamapper:"public_key" json:"public_key"`
	Address   string `gluamapper:"address" json:"address"`
}

// Configuration - a block of configuration data
// this is read from the configuration file
type Configuration struct {
	PublicIP           []string           `gluamapper:"publicip" json:"publicip"`
	NodeType           string             `gluamapper:"nodetype" json:"nodetype"`
	Port               int                `gluamapper:"port" json:"port"`
	DynamicConnections bool               `gluamapper:"dynamic_connections" json:"dynamic_connections"`
	PreferIPv6         bool               `gluamapper:"prefer_ipv6" json:"prefer_ipv6"`
	Listen             []string           `gluamapper:"listen" json:"listen"`
	Announce           []string           `gluamapper:"announce" json:"announce"`
	PrivateKey         string             `gluamapper:"private_key" json:"private_key"`
	PublicKey          string             `gluamapper:"public_key" json:"public_key"`
	Connect            []StaticConnection `gluamapper:"connect" json:"connect,omitempty"`
}

//P2PNode  A p2p node
type P2PNode struct {
	PublicKey string
	//StreamHandle *StreamHandling
	NodeType string
	Host     p2pcore.Host
	//identity     p2pcore.PeerID
	Peerstore    peerstore.Peerstore
	sync.RWMutex           // to allow locking
	log          *logger.L // logger
	// for background
	background *background.T
	// set once during initialise
	initialised bool
}

// Initialise initialize p2p module
func Initialise(configuration *Configuration, version string) error {
	globalData.Lock()
	defer globalData.Unlock()
	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}
	globalData.log = logger.New("network")
	globalData.log.Info("starting…")

	// Create A P2PNode
	globalData.Setup(configuration)
	// Node setup
	globalData.Start()
	//start background processes
	//globalData.log.Info("start background…")
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
	//globalData.background.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}
