// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	"time"
)

const (
	announceInitial     = 2 * time.Minute      // startup delay before first send
	announceRebroadcast = 7 * time.Minute      // to prevent too frequent rebroadcasts
	announceInterval    = 11 * time.Minute     // regular polling time
	announceExpiry      = 5 * announceInterval // if no responses received within this time, delete the entry
)

type announcer struct {
	log *logger.L
}

// initialise the announcer
func (ann *announcer) initialise() error {

	log := logger.New("announcer")
	ann.log = log

	log.Info("initialising…")

	return nil
}

// wait for incoming requests, process them and reply
func (ann *announcer) Run(args interface{}, shutdown <-chan struct{}) {

	log := ann.log

	log.Info("starting…")

	queue := messagebus.Bus.Announce.Chan()

	delay := time.After(announceInitial)
loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			log.Infof("received control: %s  parameters: %x", item.Command, item.Parameters)
			switch item.Command {
			case "reconnect":
				determineConnections(log)
			case "updatetime":
				if len(item.Parameters[0]) >= 2 {
					timestamp := binary.BigEndian.Uint64(item.Parameters[1])
					if timestamp != 0 {
						ts := time.Unix(int64(timestamp), 0)
						setPeerTimestamp(item.Parameters[0], ts)
					}
				}
			default:
			}

		case <-delay:
			delay = time.After(announceInterval)
			ann.process()
		}
	}
}

// process the annoucement and return response to client
func (ann *announcer) process() {

	log := ann.log

	log.Debug("process starting…")

	globalData.Lock()
	defer globalData.Unlock()

	// get a big endian timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	// announce this nodes IP and ports to other peers
	if globalData.rpcsSet {
		log.Debugf("send rpc: %x", globalData.fingerprint)
		messagebus.Bus.Broadcast.Send("rpc", globalData.fingerprint[:], globalData.rpcs, timestamp)
	}
	if globalData.peerSet {
		log.Debugf("send peer: %x", globalData.publicKey)
		messagebus.Bus.Broadcast.Send("peer", globalData.publicKey, globalData.listeners, timestamp)
	}

	expireRPC()
	expirePeer(log)

	if globalData.treeChanged {
		determineConnections(log)
		globalData.treeChanged = false
	}
}

func determineConnections(log *logger.L) {
	if nil == globalData.thisNode {
		log.Errorf("determineConnections called to early")
		return // called to early
	}

	log.Infof("DC: this: %x", globalData.publicKey)

	// N1
	n1 := globalData.thisNode.Next()
	if nil == n1 {
		n1 = globalData.peerTree.First()
	}
	if nil == n1 || n1 == globalData.thisNode {
		log.Errorf("determineConnections tree too small")
		return
	}
	peer := n1.Value().(*peerEntry)
	log.Infof("N1: peer: %s", peer)
	messagebus.Bus.Connector.Send("N1", peer.publicKey, peer.listeners)

	// N2
	node := n1.Next()
	if nil == node {
		node = globalData.peerTree.First()
	}
	if nil == node || node == globalData.thisNode {
		return // tree still too small
	}

	// N3
	n3 := node.Next()
	if nil == n3 {
		n3 = globalData.peerTree.First()
	}
	if nil == n3 || n3 == globalData.thisNode {
		return // tree still too small
	}
	if n3 != n1 {
		peer := n3.Value().(*peerEntry)
		log.Infof("N3: peer: %s", peer)
		messagebus.Bus.Connector.Send("N3", peer.publicKey, peer.listeners)
	}

	// determine X25, X50 and X75 the cross ¼,½ and ¾ positions (mod tree size)
	_, index := globalData.peerTree.Search(globalData.thisNode.Key())
	count := globalData.peerTree.Count()
	quarter := count/4 + index
	if quarter >= count {
		quarter -= count
	}

	half := count/2 + index
	if half >= count {
		half -= count
	}

	threequarters := half + count/4
	if threequarters >= count {
		threequarters -= count
	}

	log.Debugf("N0: %d  tree size: %d", index, count)
	log.Debugf("Xi: ¼: %d  ½: %d  ¾: %d", quarter, half, threequarters)

	x25 := globalData.peerTree.Get(quarter)
	x50 := globalData.peerTree.Get(half)
	x75 := globalData.peerTree.Get(threequarters)

	log.Infof("X25: this: %x", globalData.publicKey)
	if nil != x25 {
		if x25 != n1 && x25 != n3 {
			peer := x25.Value().(*peerEntry)
			log.Infof("X25: peer: %s", peer)
			messagebus.Bus.Connector.Send("X25", peer.publicKey, peer.listeners)
		}
	}
	if nil != x50 {
		if x50 != n1 && x50 != n3 && x50 != x25 {
			peer := x50.Value().(*peerEntry)
			log.Infof("X50: peer: %s", peer)
			messagebus.Bus.Connector.Send("X50", peer.publicKey, peer.listeners)
		}
	}
	if nil != x75 {
		if x75 != n1 && x75 != n3 && x75 != x25 && x75 != x50 {
			peer := x75.Value().(*peerEntry)
			log.Infof("X75: peer: %s", peer)
			messagebus.Bus.Connector.Send("X75", peer.publicKey, peer.listeners)
		}
	}
}

func expirePeer(log *logger.L) {
	now := time.Now()
	nextNode := globalData.peerTree.First()
scan_nodes:
	for node := nextNode; nil != node; node = nextNode {

		peer := node.Value().(*peerEntry)
		key := node.Key()

		nextNode = node.Next()

		// skip this node's entry
		if bytes.Equal(globalData.publicKey, peer.publicKey) {
			continue scan_nodes
		}
		log.Debugf("public key: %x timestamp: %s", peer.publicKey, peer.timestamp.Format(timeFormat))
		if peer.timestamp.Add(announceExpiry).Before(now) {
			globalData.peerTree.Delete(key)
			globalData.treeChanged = true
			messagebus.Bus.Connector.Send("@D", peer.publicKey, peer.listeners) //@D means: @->Internal Command, D->delete
			log.Infof("Peer Expired! public key: %x timestamp: %s is removed", peer.publicKey, peer.timestamp.Format(timeFormat))
		}

	}
}
