// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/gogo/protobuf/proto"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type Receptor struct {
	ID        peerlib.ID
	Listeners []ma.Multiaddr
	Timestamp time.Time // last seen time
}

// string - conversion from fmt package
func (r Receptor) String() []string {
	allAddress := make([]string, 0)
	for _, listener := range r.Listeners {
		fmt.Println("str: ", listener.String())
		allAddress = append(allAddress, listener.String())
	}
	return allAddress
}

func newPeerItem(peer *Receptor) *ReceptorPB {
	if peer == nil {
		return nil
	}
	var pbAddrs [][]byte
	for _, listener := range peer.Listeners {
		pbAddrs = append(pbAddrs, listener.Bytes())
	}
	peerIDBinary, _ := peer.ID.Marshal()
	return &ReceptorPB{
		ID:        peerIDBinary,
		Listeners: &Addrs{Address: pbAddrs},
		Timestamp: uint64(peer.Timestamp.Unix()),
	}
}

// Backup - backup all peers into peer file
func Backup(peerFile string, tree *avl.Tree) error {
	if tree.Count() <= 2 {
		return nil
	}

	peers := List{
		Receptors: make([]*ReceptorPB, 0),
	}
	lastNode := tree.Last()
	node := tree.First()

	for node != lastNode {
		peer, ok := node.Value().(*Receptor)
		if ok {
			p := newPeerItem(peer)

			peers.Receptors = append(peers.Receptors, p)
		}
		node = node.Next()
	}

	// backup the last node
	peer, ok := lastNode.Value().(*Receptor)
	if ok {
		p := newPeerItem(peer)
		peers.Receptors = append(peers.Receptors, p)
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
func Restore(peerFile string) (List, error) {
	var peers List
	data, err := ioutil.ReadFile(peerFile)
	if err != nil {
		if os.IsNotExist(err) {
			return List{}, nil
		}
		return List{}, err
	}
	err = proto.Unmarshal(data, &peers)
	if nil != err {
		return List{}, err
	}
	return peers, nil
}
