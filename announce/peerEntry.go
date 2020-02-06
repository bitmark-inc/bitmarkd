// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type peerIDkey peerlib.ID

type peerEntry struct {
	peerID    peerlib.ID
	listeners []ma.Multiaddr
	timestamp time.Time // last seen time
}

// string - conversion from fmt package
func (p peerEntry) String() []string {
	var allAddress []string
	for _, listener := range p.listeners {
		allAddress = append(allAddress, listener.String())
	}
	return allAddress
}

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
	globalData.thisNode, _ = globalData.peerTree.Search(peerIDkey(peerID))
	determineConnections(globalData.log)

	return nil
}

// isExpiredAt - is peer expired from time
func isExpiredAt(timestamp time.Time) bool {
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
	if isExpiredAt(ts) {
		return false
	}

	peer := &peerEntry{
		peerID:    peerID,
		listeners: listeners,
		timestamp: ts,
	}
	// TODO: Take care of peer update and peer replace base on protocol of multiaddress
	if node, _ := globalData.peerTree.Search(peerIDkey(peerID)); nil != node {
		peer := node.Value().(*peerEntry)

		if ts.Sub(peer.timestamp) < announceRebroadcast {
			return false
		}

	}

	// add or update the timestamp in the tree
	recordAdded := globalData.peerTree.Insert(peerIDkey(peerID), peer)

	globalData.log.Infof("Peer Added:  ID: %s,  add:%t  nodes in the peer tree: %d", peerID.String(), recordAdded, globalData.peerTree.Count())

	// if adding this nodes data
	if util.IDEqual(globalData.peerID, peerID) {
		return false
	}

	if recordAdded {
		globalData.treeChanged = true
	}

	return true
}

// resetFutureTimestampToNow - reset future timestamp to now
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

	node, _ := globalData.peerTree.Search(peerIDkey(peerID))
	if nil != node {
		node = node.Next()
	}
	if nil == node {
		node = globalData.peerTree.First()
	}
	if nil == node {
		return peerlib.ID(""), nil, time.Now(), fault.InvalidPublicKey
	}
	peer := node.Value().(*peerEntry)
	return peer.peerID, peer.listeners, peer.timestamp, nil
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
		peer := node.Value().(*peerEntry)
		if util.IDEqual(peer.peerID, globalData.peerID) || util.IDEqual(peer.peerID, peerID) {
			continue retryLoop
		}
		return peer.peerID, peer.listeners, peer.timestamp, nil
	}
	return peerlib.ID(""), nil, time.Now(), fault.InvalidPublicKey
}

// Compare - public key comparison for AVL interface
func (p peerIDkey) Compare(q interface{}) int {
	return util.IDCompare(peerlib.ID(p), peerlib.ID(q.(peerIDkey)))
}

// String - public key string convert for AVL interface
func (p peerIDkey) String() string {
	return fmt.Sprintf("%x", []byte(p))
}

// setPeerTimestamp - set the timestamp for the peer with given public key
func setPeerTimestamp(peerID peerlib.ID, timestamp time.Time) {
	globalData.Lock()
	defer globalData.Unlock()

	node, _ := globalData.peerTree.Search(peerIDkey(peerID))
	log := globalData.log
	if nil == node {
		log.Errorf("The peer with public key %x is not existing in peer tree", peerID.Pretty())
		return
	}

	peer := node.Value().(*peerEntry)
	peer.timestamp = timestamp
}
