package p2p

import (
	"context"
	"errors"
	"fmt"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/util"
	proto "github.com/golang/protobuf/proto"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	tls "github.com/libp2p/go-libp2p-tls"
	ma "github.com/multiformats/go-multiaddr"
)

//Setup setup a node
func (n *Node) Setup(configuration *Configuration, version string) error {
	globalData.Version = version
	globalData.NodeType = configuration.NodeType
	globalData.PreferIPv6 = configuration.PreferIPv6
	maAddrs := IPPortToMultiAddr(configuration.Listen)
	n.Registers = make(map[string]bool)
	prvKey, err := DecodeHexToPrvKey([]byte(configuration.PrivateKey)) //Hex Decoded binaryString
	if err != nil {
		n.Log.Error(err.Error())
		panic(err)
	}

	n.PrivateKey = prvKey
	n.NewHost(configuration.NodeType, maAddrs, n.PrivateKey)

	if n.NodeType != "Servant" {
		n.setAnnounce(configuration.Announce)
	}

	go n.listen(configuration.Announce)
	go n.MetricsNetwork.networkMonitor(n.Host, n.Log)

	//Start a block & concensus machine
	n.metricsVoting = NewMetricsPeersVoting(n)
	n.concensusMachine = NewConcensusMachine(n, &n.metricsVoting)

	//Start Broadcsting
	ps, err := pubsub.NewGossipSub(context.Background(), n.Host)
	if err != nil {
		panic(err)
	}
	n.Multicast = ps
	sub, err := n.Multicast.Subscribe(MulticastingTopic)
	go n.SubHandler(context.Background(), sub)

	globalData.initialised = true
	return nil
}

// NewHost create a NewHost according to nodetype
func (n *Node) NewHost(nodetype string, listenAddrs []ma.Multiaddr, prvKey crypto.PrivKey) error {
	options := []libp2p.Option{libp2p.Identity(prvKey), libp2p.Security(tls.ID, tls.New)}
	if "client" != nodetype {
		options = append(options, libp2p.ListenAddrs(listenAddrs...))
	}
	newHost, err := libp2p.New(context.Background(), options...)
	if err != nil {
		panic(err)
	}
	n.Host = newHost
	for _, a := range newHost.Addrs() {
		n.Log.Info(fmt.Sprintf("Host Address: %s/%v/%s\n", a, nodeProtocol, newHost.ID()))
	}
	return nil
}

//setAnnounce: Set Announce address in Routing
func (n *Node) setAnnounce(announceAddrs []string) {
	maAddrs := IPPortToMultiAddr(announceAddrs)
	fullAddr := announceMuxAddr(maAddrs, nodeProtocol, n.Host.ID())
	util.LogInfo(n.Log, util.CoLightGyan, fmt.Sprintf("setAnnounce:%v", util.PrintMaAddrs(fullAddr)))
	byteMessage, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(fullAddr)})
	param0, idErr := n.Host.ID().Marshal()
	if nil == err && nil == idErr {
		messagebus.Bus.Announce.Send("self", param0, byteMessage)
	}
}

func (n *Node) listen(announceAddrs []string) {
	maAddrs := IPPortToMultiAddr(announceAddrs)
	shandler := NewListenHandler(n.Host.ID(), n, n.Log)
	n.Host.SetStreamHandler("p2pstream", shandler.handleStream)
	n.Log.Infof("A servant is listen to %s", util.PrintMaAddrs(maAddrs))
	// Hang forever
	<-make(chan struct{})
}

func (n *Node) addRegister(id peerlib.ID) {
	n.Lock()
	n.Registers[id.Pretty()] = true
	n.Unlock()
}
func (n *Node) delRegister(id peerlib.ID) {
	n.Lock()
	n.Registers[id.Pretty()] = false
	n.Unlock()
}

func (n *Node) setConnectStatus(id peerlib.ID, status bool) {
	n.Lock()
	n.Registers[id.Pretty()] = status
	n.Unlock()
}
func (n *Node) connectStatus(id peerlib.ID) (bool, error) {
	n.Lock()
	val, ok := n.Registers[id.Pretty()]
	n.Unlock()
	if ok {
		return val, nil
	}
	return false, errors.New("peer ID does not exist")
}

//IsRegister if given id has a registered stream
func (n *Node) IsRegister(id peerlib.ID) (registered bool) {
	n.Lock()
	if isRegistered, ok := n.Registers[id.Pretty()]; ok && isRegistered {
		registered = true
	}
	n.Unlock()
	return
}
