package p2p

import (
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/p2p/concensus"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/prometheus/common/log"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
	p2pcore "github.com/libp2p/go-libp2p-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	ma "github.com/multiformats/go-multiaddr"
)

// global data
var globalData Node
var bitmarkprotocol = "/bitmark/1.0.0"

// const
const (
	// domains
	domainLocal   = "nodes.rachael.bitmark"
	domainBitamrk = "nodes.test.bitmark.com"
	domainTest    = "nodes.test.bitmark.com"
	//  time interval
	nodeInitial   = 5 * time.Second // startup delay before first send
	nodeInterval  = 1 * time.Minute // regular polling time
	lowConn       = 3
	maxConn       = 20
	connGraceTime = 30 * time.Second
)

var (
	// muticastingTopic
	multicastingTopic = "/peer/announce/1.0.0"
	// stream protocols
	nodeProtocol = ma.ProtocolWithCode(ma.P_P2P).Name
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
	NodeType           string             `gluamapper:"nodetype" json:"nodetype"`
	Port               int                `gluamapper:"port" json:"port"`
	DynamicConnections bool               `gluamapper:"dynamic_connections" json:"dynamic_connections"`
	PreferIPv6         bool               `gluamapper:"prefer_ipv6" json:"prefer_ipv6"`
	Listen             []string           `gluamapper:"listen" json:"listen"`
	Announce           []string           `gluamapper:"announce" json:"announce"`
	PrivateKey         string             `gluamapper:"private_key" json:"private_key"`
	PublicKey          string             `gluamapper:"public_key" json:"public_key"` //TODO : REMOVE
	Connect            []StaticConnection `gluamapper:"connect" json:"connect,omitempty"`
}

// NodeType to inidcate a node is a servant or client
type NodeType int

const (
	// Servant acts as both server and client
	Servant NodeType = iota
	// Client acts as a client only
	Client
	// Server acts as a server only, not supported at first draft
	Server
)

//Node  A p2p node
type Node struct {
	Version        string
	NodeType       string
	Host           p2pcore.Host
	Announce       []ma.Multiaddr
	sync.RWMutex             // to allow locking
	Log            *logger.L // logger
	RegisterStream map[string]network.Stream
	MuticastStream *pubsub.PubSub
	PreferIPv6     bool
	PrivateKey     crypto.PrivKey
	// for background
	background *background.T
	// set once during initialise
	initialised bool
	metricsNetwork
	// statemachine
	concensus.ConcensusMachine
}

// Initialise initialize p2p module
func Initialise(configuration *Configuration, version string) error {
	globalData.Lock()
	defer globalData.Unlock()
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}
	globalData.Log = logger.New("p2p")
	globalData.Log.Info("starting…")
	globalData.Setup(configuration, version)
	globalData.Log.Info("start background…")

	processes := background.Processes{
		&globalData,
		&globalData.ConcensusMachine,
	}
	globalData.background = background.Start(processes, globalData.Log)
	return nil
}

// Run  wait for incoming requests, process them and reply
func (n *Node) Run(args interface{}, shutdown <-chan struct{}) {
	log := n.Log
	log.Info("starting…")
	queue := messagebus.Bus.P2P.Chan()
	delay := time.After(nodeInitial)
	//nodeChain:= mode.ChainName()
	nodeChain := "local"
loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			log.Infof("-><- P2P received commend:%s", item.Command)
			switch item.Command {
			case "peer":
				log.Infof("-[32m<<--- ><-get peer[0m<<--- ")
				fallthrough
			case "rpc":
				if n.NodeType != "client" {
					p2pMsgPacked, err := PackP2PMessage(nodeChain, item.Command, item.Parameters)
					if err != nil {
						log.Errorf("peer command : PackP2PMessage Error")
						continue loop
					}
					err = n.MulticastWithBinaryID(p2pMsgPacked, item.Parameters[0])
					if err != nil {
						log.Errorf("Multicast Publish Error: %v\n", err)
						continue loop
					}
				}
			default:
				if "N1" == item.Command || "N3" == item.Command || "X1" == item.Command || "X2" == item.Command ||
					"X3" == item.Command || "X4" == item.Command || "X5" == item.Command || "X6" == item.Command ||
					"X7" == item.Command || "P1" == item.Command || "P2" == item.Command {
					peerID, err := peerlib.IDFromBytes(item.Parameters[0])
					log.Infof("Command:%v PeerID%s", item.Command, peerID.String())
					if err != nil {
						n.Log.Errorf("Unmarshal peer ID Error:%x", item.Parameters[0])
						continue loop
					}
					pbPeerAddrs := Addrs{}
					proto.Unmarshal(item.Parameters[1], &pbPeerAddrs)
					maAddrs := util.GetMultiAddrsFromBytes(pbPeerAddrs.Address)
					if len(maAddrs) > 0 {
						info, err := peerlib.AddrInfoFromP2pAddr(maAddrs[0])
						if err != nil {
							log.Warn(err.Error())
							continue loop
						}
						n.addPeerAddrs(*info)
						n.connectPeers()
					}
				}
			}
		case <-delay:
			delay = time.After(nodeInterval)
			log.Infof("Node Module Interval")
		}
	}
}

// Finalise - stop all background tasks
func Finalise() error {

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.Log.Info("shutting down…")
	globalData.Log.Flush()

	// stop background
	globalData.background.Stop()
	// finally...
	globalData.initialised = false

	globalData.Log.Info("finished")
	globalData.Log.Flush()

	return nil
}

//MulticastWithBinaryID muticasts packed message with given id  in binary. Use id=nil if there is no peer ID
func (n *Node) MulticastWithBinaryID(packedMessage, id []byte) error {
	if len(id) > 0 {
		err := n.MuticastStream.Publish(multicastingTopic, packedMessage)
		if err != nil {
			log.Errorf("Multicast Publish Error: %v\n", err)
			return err
		}
		displayID, err := peerlib.IDFromBytes(id)
		if err != nil {
			log.Errorf("Inavalid ID format:%v", err)
			return err
		}
		log.Infof("\x1b[32m<<--- multicasting PEER : %v\x1b[0m\n", displayID.ShortString())
	}
	log.Infof("\x1b[32m client does not broadcast to PEERs \x1b[0m\n")
	return nil
}
