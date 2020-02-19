// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"time"

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
