package p2p

import (
	"context"
	"fmt"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/gogo/protobuf/proto"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	ma "github.com/multiformats/go-multiaddr"
)

//Setup setup a node
func (n *Node) Setup(configuration *Configuration, version string) error {
	globalData.Version = version
	n.setAnnounce(configuration.Announce)
	// Start to listen to p2p stream
	go n.listen(configuration.Announce)
	// Create a Multicasting route
	ps, err := pubsub.NewGossipSub(context.Background(), n.Host)
	if err != nil {
		panic(err)
	}
	n.MuticastStream = ps
	sub, err := n.MuticastStream.Subscribe(multicastingTopic)
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
	globalData.log.Infof("New Host is created ID:%v", newHost.ID())
	for _, a := range newHost.Addrs() {
		globalData.log.Info(fmt.Sprintf("Host Address: %s/%v/%s\n", a, nodeProtocol, newHost.ID()))
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
	var shandler basicStream
	shandler.ID = fmt.Sprintf("%s", n.Host.ID())
	n.log.Infof("A servant is listen to %s", util.PrintMaAddrs(maAddrs))
	n.Host.SetStreamHandler("/chat/1.0.0", shandler.handleStream)
	// Hang forever
	<-make(chan struct{})
}
