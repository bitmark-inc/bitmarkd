package p2p

import (
	"context"
	"fmt"

	proto "github.com/golang/protobuf/proto"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	tls "github.com/libp2p/go-libp2p-tls"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

//Setup setup a node
func (n *Node) Setup(configuration *Configuration, version string, dnsPeerOnly dnsOnlyType) error {
	n.Version = version
	if nodeType(configuration.NodeType) == ClientNode {
		n.NodeType = ClientNode
	} else {
		n.NodeType = ServerNode
	}
	n.dnsPeerOnly = dnsPeerOnly
	listenIPPorts := util.DualStackAddrToIPV4IPV6(configuration.Listen)
	if len(listenIPPorts) == 0 {
		return fault.NoListenAddrs
	}
	maAddrs := util.IPPortToMultiAddr(listenIPPorts)
	n.Registers = NewRegistration(registerExpireTime)
	prvKey, err := util.DecodePrivKeyFromHex(configuration.SecretKey) //Hex Decoded binaryString
	if err != nil {
		return err
	}
	n.PrivateKey = prvKey
	n.NewHost(n.NodeType, maAddrs, n.PrivateKey)
	err = n.setAnnounce(configuration.Announce)
	if err != nil {
		if fault.NoAnnounceAddrs == err {
			n.NodeType = ClientNode
			n.Log.Info("no annouce addrs and setup a ClientNode")
		} else {
			logger.Panic(err.Error())
		}
	}
	n.listen(configuration.Announce)
	n.MetricsNetwork = NewMetricsNetwork(n.Host, n.Log)

	//Start Broadcsting
	ps, err := pubsub.NewGossipSub(context.Background(), n.Host)
	if err != nil {
		return err
	}
	n.Multicast = ps
	sub, err := n.Multicast.Subscribe(TopicMulticasting)
	if err != nil {
		return err
	}
	go n.SubHandler(context.Background(), sub)

	n.initialised = true
	return nil
}

// NewHost create a NewHost according to nodetype
func (n *Node) NewHost(nodetype nodeType, listenAddrs []ma.Multiaddr, prvKey crypto.PrivKey) error {
	options := []libp2p.Option{libp2p.Identity(prvKey), libp2p.Security(tls.ID, tls.New)}
	if ClientNode != nodetype {
		options = append(options, libp2p.ListenAddrs(listenAddrs...))
	}
	newHost, err := libp2p.New(context.Background(), options...)
	if err != nil {
		return err
	}
	n.Host = newHost
	for _, a := range newHost.Addrs() {
		n.Log.Info(fmt.Sprintf("Host Address: %s/%v/%s\n", a, nodeProtocol, newHost.ID()))
	}
	return nil
}

//setAnnounce: Set Announce address in Routing
func (n *Node) setAnnounce(announceAddrs []string) error {
	maAddrs := util.IPPortToMultiAddr(announceAddrs)
	n.Announce = n.announceFullAddr(maAddrs)
	if nil == n.Announce || 0 == len(n.Announce) {
		return fault.NoAnnounceAddrs
	}
	util.LogInfo(n.Log, util.CoReset, fmt.Sprintf("setAnnounce:%v", util.PrintMaAddrs(n.Announce)))
	byteMessage, err := proto.Marshal(&Addrs{Address: util.GetBytesFromMultiaddr(n.Announce)})
	if err != nil {
		return err
	}
	param0, idErr := n.Host.ID().Marshal()
	if idErr != nil {
		return idErr
	}
	messagebus.Bus.Announce.Send("self", param0, byteMessage)
	return nil
}

func (n *Node) listen(announceAddrs []string) {
	maAddrs := util.IPPortToMultiAddr(announceAddrs)
	shandler := NewListenHandler(n.Host.ID(), n, n.Log)
	n.Host.SetStreamHandler(protocol.ID(TopicP2P), shandler.handleStream)
	n.Log.Infof("A servant is listen to %s", util.PrintMaAddrs(maAddrs))
}

func (n *Node) announceFullAddr(ipportAnnounce []ma.Multiaddr) []ma.Multiaddr {
	maAddrs := []ma.Multiaddr{}
	p2pMa, err := ma.NewMultiaddr(fmt.Sprintf("/%s/%s", nodeProtocol, n.Host.ID().String()))
	if err != nil {
		return nil
	}

	for _, addr := range ipportAnnounce {
		maAddr := addr.Encapsulate(p2pMa)
		maAddrs = append(maAddrs, maAddr)
	}
	return maAddrs
}
