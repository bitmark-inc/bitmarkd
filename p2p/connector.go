package p2p

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	ma "github.com/multiformats/go-multiaddr"
)

//DirectConnect connect to the peer with given peer AddrInfo
func (n *Node) DirectConnect(info peer.AddrInfo) error {
	cctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	if n.isSameNode(info) { // check if the same node
		util.LogDebug(n.Log, util.CoLightGray, "DirectConnect to the self node")
		return nil
	}
	connected, _ := n.connectStatus(info.ID)
	if connected { // If connected, don't need to reconnect
		util.LogDebug(n.Log, util.CoLightGreen, fmt.Sprintf("DirectConnect ID:%v connected", info.ID.ShortString()))
		return nil
	}
	for _, addr := range info.Addrs {
		if n.PreferIPv6 && util.IsMultiAddrIPV6(addr) {
			ipv6Addr, ipv6Err := ma.NewMultiaddr(fmt.Sprintf("%s/%v/%s", addr, nodeProtocol, info.ID.ShortString()))
			ipv6Info, ipv6Err := util.MaAddrToAddrInfo(ipv6Addr)
			ipv6Err = n.Host.Connect(cctx, *ipv6Info)
			if ipv6Err == nil {
				n.setConnectStatus(info.ID, true)
				util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("DirectConnect to IPV6 addr:%v", ipv6Addr))
				_, err := n.Register(ipv6Info)
				if err == nil {
					n.addRegister(info.ID)
				} else {
					n.delRegister(info.ID)
				}
				return nil
			}
			util.LogWarn(n.Log, util.CoLightRed, fmt.Sprintf("DirectConnect to ID:%v IPV6 Error:%v", info.ID.ShortString(), ipv6Err))
		}
	}
	err := n.Host.Connect(cctx, info)
	if err != nil {
		n.setConnectStatus(info.ID, false)
		util.LogWarn(n.Log, util.CoLightRed, fmt.Sprintf("DirectConnect ID:%v Error:%v", info.ID.ShortString(), err))
		return err
	}
	n.setConnectStatus(info.ID, true)
	util.LogInfo(n.Log, util.CoGreen, fmt.Sprintf("DirectConnect to addr:%v/%v", util.PrintMaAddrs(info.Addrs), info.ID.ShortString()))
	_, err = n.Register(&info)
	if err == nil {
		n.addRegister(info.ID)
	} else {
		n.delRegister(info.ID)
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
