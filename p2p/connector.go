package p2p

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	net "github.com/libp2p/go-libp2p-net"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prometheus/common/log"
)

func (n *Node) dialPeer(ctx context.Context, remoteListner *peer.AddrInfo) (net.Stream, error) {
	cctx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()
	return n.Host.NewStream(cctx, remoteListner.ID, "/chat/1.0.0")
}

// ConnectPeers connect to all peers in host peerstore
func (n *Node) connectPeers() {

loop:
	for idx, peerID := range n.Host.Peerstore().Peers() {
		peerInfo := n.Host.Peerstore().PeerInfo(peerID)

		if len(peerInfo.Addrs) != 0 && !n.isSameNode(peerInfo.Addrs) {
			for _, addr := range peerInfo.Addrs {
				n.log.Infof("connectPeers: Dial to peer[%d]:%s", idx, addr.String())
			}
			n.directConnect(peerInfo)
		} else {
			n.log.Infof("The same node: %s", peerID)
			continue loop
		}
	}
}

func (n *Node) directConnect(info peer.AddrInfo) {
	// Start a stream with the destination.
	// Multiaddress of the destination peer is fetched from the peerstore using 'peerId'.
	s, err := n.Host.NewStream(context.Background(), info.ID, "/chat/1.0.0")
	if err != nil {
		log.Warn(err.Error())
		return
	}
	// Create a thread to read and write data.
	shandler := basicStream{ID: fmt.Sprintf("%s", n.Host.ID())}
	shandler.handleStream(s)
}

// Check on IP and Port and also local addr with the same port
func (n *Node) isSameNode(addrs []ma.Multiaddr) bool {
	for _, cmpr := range addrs {
		for _, a := range n.Announce {
			// Compare Announce Address
			if strings.Contains(cmpr.String(), a.String()) {
				n.log.Info("Peer is this NODE")
				return true
			}
		}
		// Compare local listener address
		for _, a := range n.Host.Addrs() {
			if strings.Contains(cmpr.String(), a.String()) {
				n.log.Info("Peer is this NODE")
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
			//	log.Debugf("peers in PeerStore:%s     NewAddress:%s\n", addr.String(), newAddr.String())
			if addr.Equal(newAddr) {
				n.log.Info("Peer is in PeerStore")
				return true
			}
		}
	}
	return false
}
