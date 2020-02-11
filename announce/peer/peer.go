// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bitmark-inc/bitmarkd/avl"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"

	proto "github.com/golang/protobuf/proto"
)

// NewPeerItem - create a PeerItem from receptor.Receptor
func NewPeerItem(peer *receptor.Receptor) *PeerItem {
	if peer == nil {
		return nil
	}
	var pbAddrs [][]byte
	for _, listener := range peer.Listeners {
		pbAddrs = append(pbAddrs, listener.Bytes())
	}
	peerIDBinary, _ := peer.ID.Marshal()
	return &PeerItem{
		PeerID:    peerIDBinary,
		Listeners: &Addrs{Address: pbAddrs},
		Timestamp: uint64(peer.Timestamp.Unix()),
	}
}

// Backup - backup all peers into peer file
func Backup(peerFile string, tree *avl.Tree) error {
	if tree.Count() <= 2 {
		return nil
	}

	peers := PeerList{
		Peers: make([]*PeerItem, 0),
	}
	lastNode := tree.Last()
	node := tree.First()

	for node != lastNode {
		peer, ok := node.Value().(*receptor.Receptor)
		if ok {
			p := NewPeerItem(peer)

			peers.Peers = append(peers.Peers, p)
		}
		node = node.Next()
	}

	// backup the last node
	peer, ok := lastNode.Value().(*receptor.Receptor)
	if ok {
		p := NewPeerItem(peer)
		peers.Peers = append(peers.Peers, p)
	}

	out, err := proto.Marshal(&peers)
	if nil != err {
		return fmt.Errorf("failed to marshal peer")
	}

	if err := ioutil.WriteFile(peerFile, out, 0600); err != nil {
		return fmt.Errorf("failed to write peers to a file: %s", err)
	}
	return nil
}

// Restore - restore peers from peer file
func Restore(peerFile string) (PeerList, error) {
	var peers PeerList
	data, err := ioutil.ReadFile(peerFile)
	if err != nil {
		if os.IsNotExist(err) {
			return PeerList{}, nil
		}
		return PeerList{}, err
	}
	err = proto.Unmarshal(data, &peers)
	if nil != err {
		return PeerList{}, err
	}
	return peers, nil
}
