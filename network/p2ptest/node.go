package main

import (
	"context"
	"net"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
	libp2p "github.com/libp2p/go-libp2p"
	p2pcore "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	tls "github.com/libp2p/go-libp2p-tls"
	ma "github.com/multiformats/go-multiaddr"
)

//Setup setup a node
func (p *P2PNode) Setup(configuration *Configuration) {
	globalData.NodeType = configuration.NodeType
	globalData.PublicKey = configuration.PublicKey
	globalData.Host = NewHost(globalData.NodeType, configuration.Listen, configuration.PrivateKey)
	globalData.initialised = true
}

//Start Start to connect to other nodes
func (p *P2PNode) Start() {
	p.ConnectToBootstrap()
}

// NewHost create a NewHost according to nodetype
func NewHost(nodetype string, listenAddr []string, privateKey string) p2pcore.Host {
	globalData.log.Infof("Private key:%x", []byte(privateKey))
	prvKey, err := DecodePrvKey([]byte(privateKey)) //Hex Decoded binaryString
	if err != nil {
		globalData.log.Error(err.Error())
		panic(err)
	}

	// For Client Node
	if "client" == nodetype {
		newHost, err := libp2p.New(
			context.Background(),
			libp2p.Identity(prvKey),
			libp2p.Security(tls.ID, tls.New),
		)
		if err != nil {
			panic(err)
		}
		return newHost
	}
	// For Servant Node
	var hostListenAddress ma.Multiaddr
	hostListenAddress, err = NewListenMultiAddr(listenAddr)
	newHost, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(hostListenAddress),
		libp2p.Identity(prvKey),
		libp2p.Security(tls.ID, tls.New),
	)
	if err != nil {
		panic(err)
	}
	for _, addr := range newHost.Addrs() {
		globalData.log.Infof("New Host Address: %s/%v/%s\n", addr, "p2p", newHost.ID())
	}
	return newHost
}

// ConnectToBootstrap  connect to bootstrap node
func (p *P2PNode) ConnectToBootstrap() error {
	err := lookupNodesDomain(domainLocal, globalData.log)
	return err
}

// Connect  connect to other node , this is a blocking operation
func (p *P2PNode) Connect(peer peer.AddrInfo) error {
	err := p.Host.Connect(context.Background(), peer)
	if err != nil {
		return err
	}
	for _, addr := range peer.Addrs {
		p.Host.Peerstore().AddAddr(peer.ID, addr, peerstore.ConnectedAddrTTL)
	}
	return nil
}

// lookup node domain for the peering
func lookupNodesDomain(domain string, log *logger.L) error {

	if "" == domain {
		log.Error("invalid node domain")
		return fault.ErrInvalidNodeDomain
	}

	texts, err := net.LookupTXT(domain)
	if nil != err {
		log.Errorf("lookup TXT record error: %s", err)
		return err
	}

	// process DNS entries
	for i, t := range texts {
		t = strings.TrimSpace(t)
		tag, err := parseTag(t)
		if nil != err {
			log.Errorf("ignore TXT[%d]: %q  error: %s", i, t, err)
		} else {
			log.Infof("process TXT[%d]: %q", i, t)
			log.Infof("result[%d]: IPv4: %q  IPv6: %q  rpc: %d  connect: %d", i, tag.ipv4, tag.ipv6, tag.rpcPort, tag.connectPort)
			log.Infof("result[%d]: peer public key: %x", i, tag.publicKey)
			log.Infof("result[%d]: rpc fingerprint: %x", i, tag.certificateFingerprint)

			listeners := []byte{}

			if nil != tag.ipv4 {
				c1 := util.ConnectionFromIPandPort(tag.ipv4, tag.connectPort)
				listeners = append(listeners, c1.Pack()...)
			}
			if nil != tag.ipv6 {
				c2 := util.ConnectionFromIPandPort(tag.ipv6, tag.connectPort)
				listeners = append(listeners, c2.Pack()...)
			}

			if nil == tag.ipv4 && nil == tag.ipv6 {
				log.Debugf("result[%d]: ignoring invalid record", i)
			} else {
				log.Infof("result[%d]: adding: %x", i, listeners)

				// internal add, as lock is already held
				//addPeer(tag.publicKey, listeners, uint64(time.Now().Unix()))
			}
		}
	}

	return nil
}
