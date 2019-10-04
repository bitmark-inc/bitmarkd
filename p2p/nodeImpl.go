package p2p

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/network"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/p2p/statemachine"
	"github.com/bitmark-inc/bitmarkd/util"
	proto "github.com/golang/protobuf/proto"

	libp2p "github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
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
	n.RegisterStream = make(map[string]network.Stream)
	prvKey, err := DecodeHexToPrvKey([]byte(configuration.PrivateKey)) //Hex Decoded binaryString
	if err != nil {
		n.Log.Error(err.Error())
		panic(err)
	}
	n.PrivateKey = prvKey

	n.NewHost(configuration.NodeType, maAddrs, n.PrivateKey)
	n.setAnnounce(configuration.Announce)
	go n.listen(configuration.Announce)
	go n.metricsNetwork.networkMonitor(n.Host, n.Log)

	ps, err := pubsub.NewGossipSub(context.Background(), n.Host)
	if err != nil {
		panic(err)
	}
	n.MuticastStream = ps
	sub, err := n.MuticastStream.Subscribe(multicastingTopic)
	go n.SubHandler(context.Background(), sub)

	n.stateMachine = statemachine.NewStateMachine()
	globalData.initialised = true
	return nil
}

// NewHost create a NewHost according to nodetype
func (n *Node) NewHost(nodetype string, listenAddrs []ma.Multiaddr, prvKey crypto.PrivKey) error {
	cm := connmgr.NewConnManager(lowConn, maxConn, connGraceTime)
	options := []libp2p.Option{libp2p.Identity(prvKey), libp2p.Security(tls.ID, tls.New), libp2p.ConnectionManager(cm)}
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

func (n *Node) addToRegister(id peerlib.ID, s network.Stream) {
	n.Lock()
	n.RegisterStream[id.Pretty()] = s
	n.Unlock()
}
