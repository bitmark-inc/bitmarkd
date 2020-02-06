// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/receiver"

	"github.com/bitmark-inc/bitmarkd/announce/id"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// setSelf - called by the peering initialisation to set up this
// node's announcement data
func setSelf(peerID peerlib.ID, listeners []ma.Multiaddr) error {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.peerSet {
		return fault.AlreadyInitialised
	}
	globalData.peerID = peerID
	globalData.listeners = listeners
	globalData.peerSet = true

	addPeer(peerID, listeners, uint64(time.Now().Unix()))
	globalData.thisNode, _ = globalData.peerTree.Search(id.ID(peerID))
	determineConnections(globalData.log)

	return nil
}

// isExpired - is peer expired from time
func isExpired(timestamp time.Time) bool {
	return timestamp.Add(announceExpiry).Before(time.Now())
}

// AddPeer - add a peer announcement to the in-memory tree
// returns:
//   true  if this was a new/updated entry
//   false if the update was within the limits (to prevent continuous relaying)
func AddPeer(peerID peerlib.ID, listeners []ma.Multiaddr, timestamp uint64) bool {
	globalData.Lock()
	rc := addPeer(peerID, listeners, timestamp)
	globalData.Unlock()
	return rc
}

// addPeer - internal add a peer announcement, hold lock before calling
func addPeer(peerID peerlib.ID, listeners []ma.Multiaddr, timestamp uint64) bool {
	ts := resetFutureTimestampToNow(timestamp)
	if isExpired(ts) {
		return false
	}

	r := &receiver.Receiver{
		ID:        peerID,
		Listeners: listeners,
		Timestamp: ts,
	}
	// TODO: Take care of r update and r replace base on protocol of multiaddress
	if node, _ := globalData.peerTree.Search(id.ID(peerID)); nil != node {
		peer := node.Value().(*receiver.Receiver)

		if ts.Sub(peer.Timestamp) < announceRebroadcast {
			return false
		}

	}

	// add or update the Timestamp in the tree
	recordAdded := globalData.peerTree.Insert(id.ID(peerID), r)

	globalData.log.Infof("Peer Added:  ID: %s,  add:%t  nodes in the r tree: %d", peerID.String(), recordAdded, globalData.peerTree.Count())

	// if adding this nodes data
	if util.IDEqual(globalData.peerID, peerID) {
		return false
	}

	if recordAdded {
		globalData.treeChanged = true
	}

	return true
}

// resetFutureTimestampToNow - reset future Timestamp to now
func resetFutureTimestampToNow(timestamp uint64) time.Time {
	ts := time.Unix(int64(timestamp), 0)
	now := time.Now()
	if now.Before(ts) {
		return now
	}
	return ts
}

// GetNext - fetch next node data in the ring by given public key
func GetNext(peerID peerlib.ID) (peerlib.ID, []ma.Multiaddr, time.Time, error) {
	globalData.Lock()
	defer globalData.Unlock()

	node, _ := globalData.peerTree.Search(id.ID(peerID))
	if nil != node {
		node = node.Next()
	}
	if nil == node {
		node = globalData.peerTree.First()
	}
	if nil == node {
		return peerlib.ID(""), nil, time.Now(), fault.InvalidPublicKey
	}
	peer := node.Value().(*receiver.Receiver)
	return peer.ID, peer.Listeners, peer.Timestamp, nil
}

// GetRandom - fetch random node data in the ring not matching given public key
func GetRandom(peerID peerlib.ID) (peerlib.ID, []ma.Multiaddr, time.Time, error) {
	globalData.Lock()
	defer globalData.Unlock()

retryLoop:
	for tries := 1; tries <= 5; tries += 1 {
		max := big.NewInt(int64(globalData.peerTree.Count()))
		r, err := rand.Int(rand.Reader, max)
		if nil != err {
			continue retryLoop
		}

		n := int(r.Int64()) // 0 … max-1

		node := globalData.peerTree.Get(n)
		if nil == node {
			node = globalData.peerTree.First()
		}
		if nil == node {
			break retryLoop
		}
		peer := node.Value().(*receiver.Receiver)
		if util.IDEqual(peer.ID, globalData.peerID) || util.IDEqual(peer.ID, peerID) {
			continue retryLoop
		}
		return peer.ID, peer.Listeners, peer.Timestamp, nil
	}
	return peerlib.ID(""), nil, time.Now(), fault.InvalidPublicKey
}

// setPeerTimestamp - set the timestamp for the peer with given public key
func setPeerTimestamp(peerID peerlib.ID, timestamp time.Time) {
	globalData.Lock()
	defer globalData.Unlock()

	node, _ := globalData.peerTree.Search(id.ID(peerID))
	log := globalData.log
	if nil == node {
		log.Errorf("The peer with public key %x is not existing in peer tree", peerID.Pretty())
		return
	}

	peer := node.Value().(*receiver.Receiver)
	peer.Timestamp = timestamp
}
