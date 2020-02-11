// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/peer"

	"github.com/bitmark-inc/bitmarkd/announce/receiver"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	proto "github.com/golang/protobuf/proto"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
)

const (
	//announceInitial     = 2 * time.Minute // startup delay before first send
	announceInitial = 1 * time.Minute // startup delay before first send
	//announceRebroadcast = 7 * time.Minute // to prevent too frequent rebroadcasts
	announceRebroadcast = 30 * time.Second // to prevent too frequent rebroadcasts //TODO: We may not need it anymore
	//announceInterval    = 11 * time.Minute     // regular polling time
	announceInterval = 3 * time.Minute
	//announceExpiry   = 5 * announceInterval // if no responses received within this time, delete the entry
	announceExpiry  = 5 * announceInterval
	MinTreeExpected = 5 + 1 //reference : voting minimumClients + 1(self)
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
			util.LogInfo(log, util.CoReset, fmt.Sprintf("received control: %s  parameters: %x", item.Command, item.Parameters))
			switch item.Command {
			case "reconnect":
				determineConnections(log)
			case "updatetime":
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if err != nil {
					log.Warn(err.Error())
					continue loop
				}
				setPeerTimestamp(id, time.Now())
				log.Infof("-><- updatetime id:%s", string(item.Parameters[0]))
			case "addpeer":
				//TODO: Make sure the timestamp is from external message or local timestamp
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if err != nil {
					log.Warn(err.Error())
					continue loop
				}
				timestamp := binary.BigEndian.Uint64(item.Parameters[2])
				var listeners peer.Addrs
				err = proto.Unmarshal(item.Parameters[1], &listeners)
				if err != nil {
					util.LogError(log, util.CoRed, fmt.Sprintf("addpeer: Unmarshal Address Error:%v", err))
					continue loop
				}
				addrs := util.GetMultiAddrsFromBytes(listeners.Address)
				if len(addrs) == 0 {
					util.LogError(log, util.CoRed, "No valid listener address: addrs is empty")
					continue loop
				}
				addPeer(id, addrs, timestamp)
				util.LogDebug(log, util.CoYellow, fmt.Sprintf("-><- addpeer : %s  listener: %s  Timestamp: %d", id.String(), printBinaryAddrs(item.Parameters[1]), timestamp))
				//globalData.peerTree.Print(false)
			case "addrpc":
				timestamp := binary.BigEndian.Uint64(item.Parameters[2])
				log.Infof("received rpc: fingerprint: %x  rpc: %x  Timestamp: %d", item.Parameters[0], item.Parameters[1], timestamp)
				AddRPC(item.Parameters[0], item.Parameters[1], timestamp)
			case "self":
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if err != nil {
					log.Warn(err.Error())
					continue loop
				}
				var listeners peer.Addrs
				_ = proto.Unmarshal(item.Parameters[1], &listeners)
				addrs := util.GetMultiAddrsFromBytes(listeners.Address)
				if len(addrs) == 0 {
					log.Warn("No valid listener address")
					continue loop
				}
				log.Infof("-><-  request self announce data add to tree: %v  listener: %s", id, printBinaryAddrs(item.Parameters[1]))
				setSelf(id, addrs)
			default:
			}
		case <-delay:
			delay = time.After(announceInterval)
			ann.process()
		}
	}
}

// process the annoucement and return response to receiver
func (ann *announcer) process() {

	log := ann.log

	util.LogInfo(log, util.CoReset, "process starting…")
	globalData.Lock()
	defer globalData.Unlock()

	// get a big endian Timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	// announce this nodes IP and ports to other peers
	if globalData.rpcsSet {
		log.Debugf("send rpc: %x", globalData.fingerprint)
		if globalData.dnsPeerOnly == UsePeers { //Make self a  hiden rpc node to avoid been connected
			messagebus.Bus.P2P.Send("rpc", globalData.fingerprint[:], globalData.rpcs, timestamp)
		}
	}
	if globalData.peerSet {
		addrsBinary, errAddr := proto.Marshal(&peer.Addrs{Address: util.GetBytesFromMultiaddr(globalData.listeners)})
		idBinary, errID := globalData.peerID.MarshalBinary()
		if nil == errAddr && nil == errID {
			util.LogInfo(log, util.CoYellow, fmt.Sprintf("-><-send self data to P2P ID:%v address:%v", globalData.peerID.ShortString(), util.PrintMaAddrs(globalData.listeners)))
			if globalData.dnsPeerOnly == UsePeers { //Make self a  hiden node to avoid been connected
				messagebus.Bus.P2P.Send("peer", idBinary, addrsBinary, timestamp)
			}
		}
	}
	expireRPC()
	expirePeer(log)

	//if globalData.treeChanged {
	count := globalData.peerTree.Count()
	if count <= MinTreeExpected {
		exhaustiveConnections(log)
	} else {
		determineConnections(log)
	}

	globalData.treeChanged = false
	//}
}

func determineConnections(log *logger.L) {
	if nil == globalData.thisNode {
		util.LogWarn(log, util.CoRed, fmt.Sprintf("determineConnections called to early"))
		return // called to early
	}

	// locate this node in the tree
	_, index := globalData.peerTree.Search(globalData.thisNode.Key())
	count := globalData.peerTree.Count()

	// various increment values
	e := count / 8
	q := count / 4
	h := count / 2

	jump := 3      // to deal with N3/P3 and too few nodes
	if count < 4 { // if insufficient
		jump = 1 // just duplicate N1/P1
	}

	names := [11]string{
		"N1",
		"N3",
		"X1",
		"X2",
		"X3",
		"X4",
		"X5",
		"X6",
		"X7",
		"P1",
		"P3",
	}

	// compute all possible offsets
	// if count is too small then there will be duplicate offsets
	var n [11]int
	n[0] = index + 1             // N1 (+1)
	n[1] = index + jump          // N3 (+3)
	n[2] = e + index             // X⅛
	n[3] = q + index             // X¼
	n[4] = q + e + index         // X⅜
	n[5] = h + index             // X½
	n[6] = h + e + index         // X⅝
	n[7] = h + q + index         // X¾
	n[8] = h + q + e + index     // X⅞
	n[9] = index + count - 1     // P1 (-1)
	n[10] = index + count - jump // P3 (-3)

	u := -1
deduplicate:
	for i, v := range n {
		if v == index || v == u {
			continue deduplicate
		}
		u = v
		if v >= count {
			v -= count
		}
		node := globalData.peerTree.Get(v)
		if nil != node {
			p := node.Value().(*receiver.Receiver)
			if nil != p {
				idBinary, errID := p.ID.Marshal()
				pbAddr := util.GetBytesFromMultiaddr(p.Listeners)
				pbAddrBinary, errMarshal := proto.Marshal(&peer.Addrs{Address: pbAddr})
				if nil == errID && nil == errMarshal {
					messagebus.Bus.P2P.Send(names[i], idBinary, pbAddrBinary)
					util.LogDebug(log, util.CoYellow, fmt.Sprintf("--><-- determine send to P2P %v : %s  address: %x ", names[i], p.ID.ShortString(), printBinaryAddrs(pbAddrBinary)))
				}
			}

		}
	}
}

func expirePeer(log *logger.L) {
	now := time.Now()
	nextNode := globalData.peerTree.First()
scan_nodes:
	for node := nextNode; nil != node; node = nextNode {

		peer := node.Value().(*receiver.Receiver)
		key := node.Key()

		nextNode = node.Next()

		// skip this node's entry
		if globalData.peerID.String() == peer.ID.String() {
			continue scan_nodes
		}
		if peer.Timestamp.Add(announceExpiry).Before(now) {
			globalData.peerTree.Delete(key)
			globalData.treeChanged = true
			util.LogDebug(log, util.CoReset, fmt.Sprintf("expirePeer : PeerID: %v! Timestamp: %s", peer.ID.ShortString(), peer.Timestamp.Format(timeFormat)))
			idBinary, errID := peer.ID.Marshal()
			if nil == errID {
				messagebus.Bus.P2P.Send("@D", idBinary)
				util.LogInfo(log, util.CoYellow, fmt.Sprintf("--><-- Send @D to P2P  PeerID: %v", peer.ID.ShortString()))
			}
		}

	}
}

func exhaustiveConnections(log *logger.L) {
	if nil == globalData.thisNode {
		util.LogWarn(log, util.CoRed, fmt.Sprintf("exhaustiveConnections called to early"))
		return // called to early
	}
	// locate this node in the tree
	count := globalData.peerTree.Count()
	for i := 0; i < count; i++ {
		node := globalData.peerTree.Get(i)
		if nil != node {
			p := node.Value().(*receiver.Receiver)
			if nil != p && !util.IDEqual(p.ID, globalData.peerID) {
				idBinary, errID := p.ID.Marshal()
				pbAddr := util.GetBytesFromMultiaddr(p.Listeners)
				pbAddrBinary, errMarshal := proto.Marshal(&peer.Addrs{Address: pbAddr})
				if nil == errID && nil == errMarshal {
					messagebus.Bus.P2P.Send("ES", idBinary, pbAddrBinary)
					util.LogDebug(log, util.CoYellow, fmt.Sprintf("--><-- exhaustiveConnections send to P2P %v : %s  address: %x ", "ES", p.ID.ShortString(), printBinaryAddrs(pbAddrBinary)))
				}
			}
		}
	}
	// locate this node in the tree
}
