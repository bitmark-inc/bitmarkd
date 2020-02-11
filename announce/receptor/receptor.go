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

func newReceptor(r *Receptor) *ReceptorPB {
	if r == nil {
		return nil
	}
	var pbAddrs [][]byte
	for _, listener := range r.Listeners {
		pbAddrs = append(pbAddrs, listener.Bytes())
	}
	binaryID, _ := r.ID.Marshal()
	return &ReceptorPB{
		ID:        binaryID,
		Listeners: &Addrs{Address: pbAddrs},
		Timestamp: uint64(r.Timestamp.Unix()),
	}
}

// Backup - backup all receptors into file
func Backup(backupFile string, tree *avl.Tree) error {
	if tree.Count() <= 2 {
		return nil
	}

	peers := List{
		Receptors: make([]*ReceptorPB, 0),
	}
	lastNode := tree.Last()
	node := tree.First()

	// TODO: refactor
	for node != lastNode {
		peer, ok := node.Value().(*Receptor)
		if ok {
			p := newReceptor(peer)

			peers.Receptors = append(peers.Receptors, p)
		}
		node = node.Next()
	}

	// backup the last node
	peer, ok := lastNode.Value().(*Receptor)
	if ok {
		p := newReceptor(peer)
		peers.Receptors = append(peers.Receptors, p)
	}

	out, err := proto.Marshal(&peers)
	if nil != err {
		return fmt.Errorf("failed to marshal receptor")
	}

	if err := ioutil.WriteFile(backupFile, out, 0600); err != nil {
		return fmt.Errorf("failed writing receptors to file: %s", err)
	}
	return nil
}

// Restore - restore receptors from file
func Restore(backupFile string) (List, error) {
	var list List
	data, err := ioutil.ReadFile(backupFile)
	if err != nil {
		if os.IsNotExist(err) {
			return List{}, nil
		}
		return List{}, err
	}
	err = proto.Unmarshal(data, &list)
	if nil != err {
		return List{}, err
	}
	return list, nil
}
