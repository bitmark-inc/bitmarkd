package p2p

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
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

// const
const (
	//  time interval
	nodeInitial        = 5 * time.Second // startup delay before first send
	nodeInterval       = 2 * time.Minute // regular
	registerExpireTime = 2 * time.Minute
	connectCancelTime  = 30 * time.Second
)

var (
	// TopicMulticasting is the topic for multicasting for gossip multicast
	TopicMulticasting = "/multicast/1.0.0"
	// TopicP2P is the topic for p2p stream
	TopicP2P = "p2pstream"
	//nodeProtocol
	nodeProtocol = ma.ProtocolWithCode(ma.P_P2P).Name
)

type dnsOnlyType bool

const (
	DnsOnly  dnsOnlyType = true
	UsePeers dnsOnlyType = false
)

type nodeType string

const (
	ServerNode nodeType = "server"
	ClientNode nodeType = "client"
)

func (t nodeType) String() string {
	if t == ClientNode {
		return string(ClientNode)
	}
	return string(ServerNode)
}

// StaticConnection - hardwired connections
// this is read from the configuration file
type StaticConnection struct {
	PublicKey string `gluamapper:"public_key" json:"public_key"`
	Address   string `gluamapper:"address" json:"address"`
}

// Configuration - a block of configuration data
// this is read from the configuration file
type Configuration struct {
	NodeType   string             `gluamapper:"nodetype" json:"nodetype"`
	Port       int                `gluamapper:"port" json:"port"`
	Listen     []string           `gluamapper:"listen" json:"listen"`
	Announce   []string           `gluamapper:"announce" json:"announce"`
	PrivateKey string             `gluamapper:"private_key" json:"private_key"`
	Connect    []StaticConnection `gluamapper:"connect" json:"connect,omitempty"`
}

// RegisterStatus is the struct to reflect the register status of a node
type RegisterStatus struct {
	Registered   bool
	RegisterTime time.Time
}

//Node  A p2p node
type Node struct {
	Version      string
	NodeType     nodeType
	Host         p2pcore.Host
	Announce     []ma.Multiaddr
	sync.RWMutex           // to allow locking
	Log          *logger.L // logger
	Registers    map[peerlib.ID]RegisterStatus
	Multicast    *pubsub.PubSub
	PrivateKey   crypto.PrivKey
	// for background
	background *background.T
	// set once during initialise
	initialised bool
	*MetricsNetwork
	dnsPeerOnly dnsOnlyType
}

// Connected - representation of a connected Peer (For Http RPC)
type Connected struct {
	Address []string `json:"address"`
	Server  string   `json:"server"`
}

// Initialise initialize p2p module
func Initialise(configuration *Configuration, version string, dnsPeerOnly dnsOnlyType) error {
	globalData.Lock()
	defer globalData.Unlock()
	if globalData.initialised {
		return fault.AlreadyInitialised
	}
	globalData.Log = logger.New("p2p")

	globalData.Log.Info("starting…")
	globalData.Setup(configuration, version, dnsPeerOnly)
	globalData.Log.Info("start background…")

	processes := background.Processes{
		&globalData,
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
	nodeChain := mode.ChainName()
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
				if len(id) > 0 {
					displayID, err := peerlib.IDFromBytes(id)
					if nil != err {
						util.LogInfo(log, util.CoGreen, fmt.Sprintf("@D parse id Error %v", err))
					}
					n.delRegister(displayID)
					util.LogInfo(log, util.CoWhite, fmt.Sprintf("@D  ID:%v is deleted", displayID.ShortString()))
				}
			case "peer", "rpc": // only server broadcast its peer and rpc
				if ClientNode == n.NodeType {
					break
				}
				fallthrough
			case "block", "proof", "transfer", "issues", "assets":
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
					if len(id) > 0 {
						displayID, err := peerlib.IDFromBytes(id)
						if nil == err {
							util.LogInfo(log, util.CoGreen, fmt.Sprintf("<<-- multicasting PEER : %v", displayID.ShortString()))
						}
					}
				} else {
					util.LogInfo(log, util.CoGreen, fmt.Sprintf("<<--Multicast Command:%s parameters:%d\n", item.Command, len(item.Parameters)))
				}

			//general broadcasting
			default: //peers to connect
				if "N1" == item.Command || "N3" == item.Command || "X1" == item.Command || "X2" == item.Command ||
					"X3" == item.Command || "X4" == item.Command || "X5" == item.Command || "X6" == item.Command ||
					"X7" == item.Command || "P1" == item.Command || "P2" == item.Command || "ES" == item.Command {
					peerID, err := peerlib.IDFromBytes(item.Parameters[0])
					util.LogInfo(n.Log, util.CoYellow, fmt.Sprintf("Recieve Command:%v ID:%v", item.Command, peerID.ShortString()))
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
			util.LogDebug(n.Log, util.CoMagenta, fmt.Sprintf("@@NumOfGoRoutine:%d", runtime.NumGoroutine()))
			go n.updateRegistersExpiry()
		}
	}
}

// Finalise - stop all background tasks
func Finalise() error {
	if !globalData.initialised {
		return fault.NotInitialised
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

//GlobalP2PNode return p2p node for other packages to use
func GlobalP2PNode() *Node {
	return &globalData
}

//MulticastCommand muticasts packed message with given id  in binary. Use id=nil if there is no peer ID
func MulticastCommand(packedMessage []byte) error {
	err := globalData.Multicast.Publish(TopicMulticasting, packedMessage)
	if err != nil {
		util.LogWarn(globalData.Log, util.CoLightRed, fmt.Sprintf("MulticastCommand Publish error:%v", err))
		return err
	}
	return nil
}

//ID return this node host ID
func ID() peerlib.ID {
	return globalData.Host.ID()
}

// GetAllPeers - obtain a list of all connector clients
func GetAllPeers() []*Connected {
	var peers []*Connected
	for key, status := range globalData.Registers {
		if status.Registered && globalData.MetricsNetwork.IsConnected(key) {
			addrInfo := globalData.Host.Peerstore().PeerInfo(key)
			addrs := []string{}
			for _, addr := range addrInfo.Addrs {
				addrs = append(addrs, addr.String())
			}
			peers = append(peers, &Connected{Server: addrInfo.ID.String(), Address: addrs})
		}
	}
	return peers
}
