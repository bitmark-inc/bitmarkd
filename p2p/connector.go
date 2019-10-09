package p2p

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// ConnectPeers connect to all peers in host peerstore
func (n *Node) connectPeers() {
loop:
	for idx, peerID := range n.Host.Peerstore().PeersWithAddrs() {
		peerInfo := n.Host.Peerstore().PeerInfo(peerID)
		n.Log.Infof("connect to peer[%s] %s... ", peerInfo.ID, util.PrintMaAddrs(peerInfo.Addrs))
		if len(peerInfo.Addrs) == 0 {
			n.Log.Infof("no Addr: %s", peerID)
			continue loop
		} else if n.isSameNode(peerInfo) {
			n.Log.Infof("The same node: %s", peerID)
			continue loop
		} else {
			for _, addr := range peerInfo.Addrs {
				n.Log.Infof("connectPeers: Dial to peer[%d]:%s", idx, addr.String())
			}
			err := n.DirectConnect(peerInfo)
			if err != nil {
				continue loop
			}
			_, err = n.Register(&peerInfo)
			if err != nil {
				n.Log.Warn(fmt.Sprintf(":\x1b[31mRegister Failed: %v:\x1b[0m", err))
				n.Host.Network().ClosePeer(peerInfo.ID)
				continue loop
			}
		}
	}
}

//DirectConnect connect to the peer with given peer AddrInfo
func (n *Node) DirectConnect(info peer.AddrInfo) error {
	cctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	err := n.Host.Connect(cctx, info)
	if err != nil {
		n.Log.Warn(err.Error())
		return err
	}
	return nil
}

// Check on IP and Port and also local addr with the same port
func (n *Node) isSameNode(info peer.AddrInfo) bool {
	if n.Host.ID().Pretty() == info.ID.Pretty() {
		return true
	}
	for _, cmpr := range info.Addrs {
		for _, a := range n.Announce {
			// Compare Announce Address
			if strings.Contains(cmpr.String(), a.String()) {
				return true
			}
		}
		// Compare local listener address
		for _, a := range n.Host.Addrs() {
			if strings.Contains(cmpr.String(), a.String()) {
				return true
			}
		}
	}
	return false
}

//IsPeerExisted peer is existed in the Peerstore
func (n *Node) IsPeerExisted(newAddr multiaddr.Multiaddr) bool {
	//TODO: refactor nested loop
	for _, ID := range n.Host.Peerstore().Peers() {
		for _, addr := range n.Host.Peerstore().PeerInfo(ID).Addrs {
			//	Log.Debugf("peers in PeerStore:%s     NewAddress:%s\n", addr.String(), newAddr.String())
			if addr.Equal(newAddr) {
				n.Log.Info("Peer is in PeerStore")
				return true
			}
		}
	}
	return false
}
