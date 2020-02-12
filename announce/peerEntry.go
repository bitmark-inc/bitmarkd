// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"

	"github.com/bitmark-inc/bitmarkd/announce/id"

	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// AddPeer - add a peer announcement to the in-memory tree
// returns:
//   true  if this was a new/updated entry
//   false if the update was within the limits (to prevent continuous relaying)
func AddPeer(peerID peerlib.ID, listeners []ma.Multiaddr, timestamp uint64) bool {
	return globalData.receptors.Add(peerID, listeners, timestamp)
}

// GetNext - fetch next node data in the ring by given public key
func GetNext(peerID peerlib.ID) (peerlib.ID, []ma.Multiaddr, time.Time, error) {
	return globalData.receptors.Next(peerID)
}

// GetRandom - fetch random node data in the ring not matching given public key
func GetRandom(peerID peerlib.ID) (peerlib.ID, []ma.Multiaddr, time.Time, error) {
	return globalData.receptors.Random(peerID)
}

// setPeerTimestamp - set the timestamp for the peer with given public key
// TODO: move into receptor
func setPeerTimestamp(peerID peerlib.ID, timestamp time.Time) {
	globalData.Lock()
	defer globalData.Unlock()

	node, _ := globalData.receptors.Tree().Search(id.ID(peerID))
	log := globalData.log
	if nil == node {
		log.Errorf("The peer with public key %x is not existing in peer tree", peerID.Pretty())
		return
	}

	peer := node.Value().(*receptor.Data)
	peer.Timestamp = timestamp
}
