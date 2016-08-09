// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	"time"
)

const (
	announceInitial     = 2 * time.Minute
	announceRebroadcast = 8 * time.Minute // to prevent too frequent rebroadcasts
	announceInterval    = 10 * time.Minute
	announceExpiry      = 60 * time.Minute
)

type announcer struct {
	log *logger.L
}

// initialise the announcer
func (ann *announcer) initialise() error {

	log := logger.New("announcer")
	if nil == log {
		return fault.ErrInvalidLoggerChannel
	}
	ann.log = log

	log.Info("initialising…")

	return nil
}

// wait for incoming requests, process them and reply
func (ann *announcer) Run(args interface{}, shutdown <-chan struct{}) {

	log := ann.log

	log.Info("starting…")

	delay := time.After(announceInitial)
loop:
	for {
		log.Info("waiting…")
		select {
		case <-shutdown:
			break loop
		case <-delay:
			delay = time.After(announceInterval)
			ann.process()
		}
	}
}

// process the ann and return response to client
func (ann *announcer) process() {

	log := ann.log

	log.Info("process starting…")

	globalData.Lock()
	defer globalData.Unlock()

	// announce this nodes IP and ports to other peers
	if globalData.rpcsSet {
		messagebus.Bus.Broadcast.Send("rpc", globalData.fingerprint[:], globalData.rpcs)
	}
	if globalData.peerSet {
		messagebus.Bus.Broadcast.Send("peer", globalData.publicKey, globalData.broadcasts, globalData.listeners)
	}

	if globalData.change {
		determineConnections(log)
		globalData.change = false
	}
}

func determineConnections(log *logger.L) {
	if nil == globalData.thisNode {
		log.Errorf("determineConnections called to early")
		return // called to early
	}

	// N1
	node := globalData.thisNode.Next()
	if nil == node {
		node = globalData.peerTree.First()
	}
	if nil == node || node == globalData.thisNode {
		log.Errorf("determineConnections tree too small")
		return // tree still too small
	}
	if globalData.n1 != node {
		globalData.n1 = node
		peer := node.Value().(*peerEntry)
		log.Infof("N1: this: %x", globalData.publicKey)
		log.Infof("N1: peer: %x", peer)
		messagebus.Bus.Subscriber.Send("N1", peer.publicKey, peer.broadcasts)
		messagebus.Bus.Connector.Send("N1", peer.publicKey, peer.listeners)
	}

	// N2
	node = node.Next()
	if nil == node {
		node = globalData.peerTree.First()
	}
	if nil == node || node == globalData.thisNode {
		return // tree still too small
	}

	// N3
	node = node.Next()
	if nil == node {
		node = globalData.peerTree.First()
	}
	if nil == node || node == globalData.thisNode {
		return // tree still too small
	}
	if globalData.n3 != node {
		globalData.n3 = node
		peer := node.Value().(*peerEntry)
		log.Infof("N3: this: %x", globalData.publicKey)
		log.Infof("N3: peer: %x", peer)
		messagebus.Bus.Subscriber.Send("N3", peer.publicKey, peer.broadcasts)
		messagebus.Bus.Connector.Send("N3", peer.publicKey, peer.listeners)
	}

	// ***** FIX THIS: more code to determine X25, X50 and X75 the cross ¼,½ and ¾ positions
	// ***** FIX THIS:   possible treat key as a number and compute; assuming uniformly distributed keys
	// ***** FIX THIS:   but would need the tree search to be able to find the "next highest/lowest key" for this to work
	// ***** FIX THIS: more code to determine some random positions
}
