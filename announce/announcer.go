// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"fmt"
	"math/rand"

	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	"time"
)

const (
	announceInitial     = 2 * time.Minute  // startup delay be for first send
	announceRebroadcast = 7 * time.Minute  // to prevent too frequent rebroadcasts
	announceInterval    = 11 * time.Minute // regular polling time
	announceExpiry      = 70 * time.Minute // if no responses received within this time, delete the entry

	broadcastCount = 5 // how many peer a node is going to braodcast each time
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
		treeCount := globalData.peerTree.Count()
		lastPeer := globalData.lastBroadcastPeer
		var iterNum int
		for i := 0; i < broadcastCount; i++ {
			iterNum = (lastPeer + i) % treeCount
			if iterNum == lastPeer && i != 0 {
				iterNum -= 1
				break
			}

			treeRoot := globalData.peerTree.Root()
			node := treeRoot.GetNodeByOrder(uint(iterNum))
			peer := node.Value().(*peerEntry)
			log.Debugf("Current iter no. is : %d. broadcasting: %x", iterNum, peer.publicKey)
			messagebus.Bus.Broadcast.Send("peer", peer.publicKey, peer.broadcasts, peer.listeners)
		}
		globalData.lastBroadcastPeer = iterNum + 1
	}

	if globalData.change {
		determineConnections(log)
		globalData.change = false
	}
	expireRPC()
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
	thisNode := globalData.thisNode
	nodeDepth := thisNode.Depth()
	treeRoot := globalData.peerTree.Root()
	lv2NodeChildren := treeRoot.GetChildrenByDepth(2)

	toConnectTree := make([]*avl.Node, 0, 3)
	toConnectNode := make([]*avl.Node, 0, 3)
	var connectOrder uint

	if nodeDepth < 2 {
		if len(lv2NodeChildren) > 3 {
			switch thisNode.Key().Compare(treeRoot.Key()) {
			case -1:
				toConnectTree = lv2NodeChildren[:3]
			case 1:
				fallthrough
			case 0:
				toConnectTree = lv2NodeChildren[1:]
			}
		} else {
			toConnectTree = lv2NodeChildren
		}
		connectOrder = uint(rand.Uint32())
	} else if nodeDepth >= 2 {
		depth2Parent := thisNode
		// find parent node in level 2 by search parent recursively
		for l := nodeDepth; l > 2; l-- {
			depth2Parent = depth2Parent.Parent()
		}

		// try to find rest of nodes which is not an ancestor in level 2
		for _, n := range lv2NodeChildren {
			if n.Key().Compare(depth2Parent.Key()) != 0 {
				toConnectTree = append(toConnectTree, n)
			}
		}
		connectOrder = depth2Parent.GetOrder(thisNode.Key())
	}

	for _, n := range toConnectTree {
		toConnectNode = append(toConnectNode, n.GetNodeByOrder(connectOrder))
	}

	for i, node := range toConnectNode {
		nodeLabel := fmt.Sprintf("X%d", (i+1)*25) // it should by X25, X50 and X75
		if nil == node {
			log.Warnf("The node of %s is nil. This should not be happended.", nodeLabel)
			continue
		}

		if node == globalData.thisNode || node == globalData.n1 || node == globalData.n3 {
			continue
		}

		if n := globalData.crossNodes[nodeLabel]; n != node {
			globalData.crossNodes[nodeLabel] = node
			peer := node.Value().(*peerEntry)
			log.Infof("%s: this: %x", nodeLabel, globalData.publicKey)
			log.Infof("%s: peer: %x", nodeLabel, peer)
			messagebus.Bus.Subscriber.Send(nodeLabel, peer.publicKey, peer.broadcasts)
			messagebus.Bus.Connector.Send(nodeLabel, peer.publicKey, peer.listeners)
		}
	}
	// ***** FIX THIS:   possible treat key as a number and compute; assuming uniformly distributed keys
	// ***** FIX THIS:   but would need the tree search to be able to find the "next highest/lowest key" for this to work
	// ***** FIX THIS: more code to determine some random positions
}
