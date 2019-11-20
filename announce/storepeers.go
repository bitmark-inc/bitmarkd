// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"io/ioutil"
	"os"

	"github.com/bitmark-inc/bitmarkd/util"

	proto "github.com/golang/protobuf/proto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

// NewPeerItem is to create a PeerItem from peerEntry
func NewPeerItem(peer *peerEntry) *PeerItem {
	if peer == nil {
		return nil
	}
	var pbAddrs [][]byte
	for _, listner := range peer.listeners {
		pbAddrs = append(pbAddrs, listner.Bytes())
	}
	peerIDBinary, err := peer.peerID.Marshal()
	if err != nil {
		return nil
	}
	return &PeerItem{
		PeerID:    peerIDBinary,
		Listeners: &Addrs{Address: pbAddrs},
		Timestamp: uint64(peer.timestamp.Unix()),
	}
}

// storePeers will backup all peers into a peer file
func storePeers(peerFile string) error {
	if globalData.peerTree.Count() <= 2 {
		globalData.log.Info("no need to backup. peer nodes are less than two")
		return nil
	}
	var peers PeerList
	lastNode := globalData.peerTree.Last()
	node := globalData.peerTree.First()

	for node != lastNode {
		peer, ok := node.Value().(*peerEntry)
		if ok {
			p := NewPeerItem(peer)
			peers.Peers = append(peers.Peers, p)
		}
		node = node.Next()
	}
	// backup the last node
	peer, ok := lastNode.Value().(*peerEntry)
	if ok {
		p := NewPeerItem(peer)
		peers.Peers = append(peers.Peers, p)
	}
	out, err := proto.Marshal(&peers)
	if err != nil {
		globalData.log.Errorf("Failed to marshal peers protobuf:%v", err)
	}
	if err := ioutil.WriteFile(peerFile, out, 0600); err != nil {
		globalData.log.Errorf("Failed to write peers to a file:%v", err)
	}
	return nil
}

// restorePeers will backup peers from a peer file
func restorePeers(peerFile string) (PeerList, error) {
	var peers PeerList
	readin, err := ioutil.ReadFile(peerFile)
	if err != nil {
		if os.IsNotExist(err) {
			return PeerList{}, nil
		}
		globalData.log.Errorf("Failed to read peers from a file:%v", err)
		return PeerList{}, err
	}
	proto.Unmarshal(readin, &peers)
loop:
	for _, peer := range peers.Peers {
		id, err := peerlib.IDFromBytes(peer.PeerID)
		maAddrs := util.GetMultiAddrsFromBytes(peer.Listeners.Address)
		if err != nil || nil != maAddrs {
			continue loop
		}
		addPeer(id, maAddrs, peer.Timestamp)
		globalData.peerTree.Print(false)
	}
	return peers, nil
}
