package p2p

import (
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
	p2pcore "github.com/libp2p/go-libp2p-core"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
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
	nodeInitial        = 5 * time.Second // startup delay before first send
	nodeInterval       = 2 * time.Minute // regular
	lowConn            = 3
	maxConn            = 12
	connGraceTime      = 30 * time.Second
	registerExpireTime = 5 * time.Minute
	connectCancelTime  = 60 * time.Second
)

var (
	//MulticastingTopic
	MulticastingTopic = "/multicast/1.0.0"
	//nodeProtocol
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

type RegisterStatus struct {
	Registered   bool
	RegisterTime time.Time
}

//Node  A p2p node
type Node struct {
	Version       string
	NodeType      string
	Host          p2pcore.Host
	Announce      []ma.Multiaddr
	sync.RWMutex            // to allow locking
	Log           *logger.L // logger
	Registers     map[peerlib.ID]RegisterStatus
	ConnectStatus map[peerlib.ID]bool
	Multicast     *pubsub.PubSub
	PreferIPv6    bool
	PrivateKey    crypto.PrivKey
	// for background
	background *background.T
	// set once during initialise
	initialised bool
	MetricsNetwork
	metricsVoting MetricsPeersVoting
	// statemachine
	concensusMachine Machine
}

// Connected - representation of a connected Peer (For Http RPC)
type Connected struct {
	Address []string `json:"address"`
	Server  string   `json:"server"`
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
		&globalData.concensusMachine,
		&globalData.metricsVoting,
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
			switch item.Command {
			case "@D":
				if len(item.Parameters) != 1 {
					util.LogWarn(log, util.CoLightRed, fmt.Sprintf("@D parameter != 1"))
					continue loop
				}
				id := item.Parameters[0]
				if id != nil && len(id) > 0 {
					displayID, err := peerlib.IDFromBytes(id)
					if nil != err {
						util.LogInfo(log, util.CoGreen, fmt.Sprintf("@D parse id Error %v", err))
					}
					n.delRegister(displayID)
					util.LogInfo(log, util.CoCyan, fmt.Sprintf("@D  ID:%v is deleted", displayID.ShortString()))
				}
			case "peer": // only servant broadcast its peer and rpc
				fallthrough
			case "rpc":
				fallthrough
			case "block":
				fallthrough
			case "proof":
				fallthrough
			case "transfer":
				fallthrough
			case "issues":
				fallthrough
			case "assets":
				if n.NodeType != "client" {
					p2pMsgPacked, err := PackP2PMessage(nodeChain, item.Command, item.Parameters)
					if err != nil {
						util.LogWarn(log, util.CoLightRed, fmt.Sprintf("Run:PackP2PMessage error:%v", err))
						continue loop
					}
					err = MulticastCommand(p2pMsgPacked)
					if err != nil {
						util.LogWarn(log, util.CoLightRed, fmt.Sprintf("Run:Multicast Publish error:%v", err))
						continue loop
					}
					if item.Command == "peer" {
						id := item.Parameters[0]
						if id != nil && len(id) > 0 {
							displayID, err := peerlib.IDFromBytes(id)
							if nil == err {
								util.LogInfo(log, util.CoGreen, fmt.Sprintf("<<-- multicasting PEER : %v", displayID.ShortString()))
							}
						}
					} else {
						util.LogInfo(log, util.CoGreen, fmt.Sprintf("<<--Multicast Command:%s parameters:%d\n", item.Command, len(item.Parameters)))
					}
				}
			//general broadcasting
			default: //peers to connect
				if "N1" == item.Command || "N3" == item.Command || "X1" == item.Command || "X2" == item.Command ||
					"X3" == item.Command || "X4" == item.Command || "X5" == item.Command || "X6" == item.Command ||
					"X7" == item.Command || "P1" == item.Command || "P2" == item.Command {
					peerID, err := peerlib.IDFromBytes(item.Parameters[0])
					util.LogDebug(n.Log, util.CoYellow, fmt.Sprintf("Recieve Command:%v ID:%v", item.Command, peerID.ShortString()))
					if err != nil {
						util.LogWarn(log, util.CoLightRed, fmt.Sprintf("Unmarshal peer ID error:%x", item.Parameters[0]))
						continue loop
					}
					pbPeerAddrs := Addrs{}
					err = proto.Unmarshal(item.Parameters[1], &pbPeerAddrs)
					if err != nil {
						util.LogWarn(log, util.CoLightRed, fmt.Sprintf("Unmarshal  Errorr:%x Error:%v", item.Parameters[0], err))
						continue loop
					}
					maAddrs := util.GetMultiAddrsFromBytes(pbPeerAddrs.Address)
					if len(maAddrs) > 0 {
						info, err := peerlib.AddrInfoFromP2pAddr(maAddrs[0])
						info.ID = peerID
						if err != nil {
							util.LogWarn(log, util.CoLightRed, fmt.Sprintf("peer Address error:%v", err))
							continue loop
						}
						n.DirectConnect(*info)
					} else {
						util.LogWarn(log, util.CoLightRed, fmt.Sprintf("peer Address length:%d", len(maAddrs)))
					}
				} // ignore if command is not one of it ie. "ignore:"
			}
		case <-delay:
			delay = time.After(nodeInterval) // periodical process
			go n.updateRegistersExpiry()
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

//MulticastCommand muticasts packed message with given id  in binary. Use id=nil if there is no peer ID
func MulticastCommand(packedMessage []byte) error {
	err := globalData.Multicast.Publish(MulticastingTopic, packedMessage)
	if err != nil {
		util.LogWarn(globalData.Log, util.CoLightRed, fmt.Sprintf("MulticastCommand Publish error:%v", err))
		return err
	}
	return nil
}

// BlockHeight - return global block height
func BlockHeight() uint64 {
	return globalData.concensusMachine.electedHeight
}

//ID return this node host ID
func ID() peerlib.ID {
	return globalData.Host.ID()
}

// GetAllPeers - obtain a list of all connector clients
func GetAllPeers() []*Connected {
	//info := []Connected{}
	globalData.RLock()
	var peers []*Connected
	for key, status := range globalData.Registers {
		if status.Registered {
			addrInfo := globalData.Host.Peerstore().PeerInfo(key)
			addrs := []string{}
			for _, addr := range addrInfo.Addrs {
				addrs = append(addrs, addr.String())
			}
			peers = append(peers, &Connected{Server: addrInfo.ID.String(), Address: addrs})
		}
	}
	globalData.RUnlock()
	return peers
}
