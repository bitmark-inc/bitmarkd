// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	"github.com/golang/protobuf/proto"
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
func (ann *announcer) Run(_ interface{}, shutdown <-chan struct{}) {
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
				globalData.receptors.BalanceTree()
			case "updatetime":
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if err != nil {
					log.Warn(err.Error())
					continue loop
				}
				globalData.receptors.UpdateTime(id, time.Now())
				log.Infof("-><- updatetime id:%s", string(item.Parameters[0]))
			case "addpeer":
				//TODO: Make sure the timestamp is from external message or local timestamp
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if err != nil {
					log.Warn(err.Error())
					continue loop
				}
				timestamp := binary.BigEndian.Uint64(item.Parameters[2])
				var listeners receptor.Addrs
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
				globalData.receptors.Add(id, addrs, timestamp)
				util.LogDebug(log, util.CoYellow, fmt.Sprintf("-><- addpeer : %s  listener: %s  Timestamp: %d", id.String(), receptor.AddrToString(item.Parameters[1]), timestamp))
				//globalData.tree.Print(false)
			case "addrpc":
				timestamp := binary.BigEndian.Uint64(item.Parameters[2])
				log.Infof("received rpc: fingerprint: %x  rpc: %x  Timestamp: %d", item.Parameters[0], item.Parameters[1], timestamp)
				addRPC(item.Parameters[0], item.Parameters[1], timestamp)
			case "self":
				id, err := peerlib.IDFromBytes(item.Parameters[0])
				if err != nil {
					log.Warn(err.Error())
					continue loop
				}
				var listeners receptor.Addrs
				_ = proto.Unmarshal(item.Parameters[1], &listeners)
				addrs := util.GetMultiAddrsFromBytes(listeners.Address)
				if len(addrs) == 0 {
					log.Warn("No valid listener address")
					continue loop
				}
				log.Infof("-><-  request self announce data add to tree: %v  listener: %s", id, receptor.AddrToString(item.Parameters[1]))
				err = globalData.receptors.SetSelf(id, addrs)
				if nil != err {
					log.Errorf("announcer set with error: %s", err)
				}
			default:
			}
		case <-delay:
			delay = time.After(announceInterval)
			ann.process()
		}
	}
}

// process the announcement and return response to receptor
func (ann *announcer) process() {
	log := ann.log

	util.LogInfo(log, util.CoReset, "process starting…")
	globalData.Lock()
	defer globalData.Unlock()

	// get a big endian Timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	// announce this nodes IP and ports to other peers
	if globalData.rpcs.IsSet() {
		log.Debugf("send rpc: %x", globalData.fingerprint)
		if globalData.dnsPeerOnly == UsePeers { //Make self a  hiden rpc node to avoid been connected
			messagebus.Bus.P2P.Send("rpc", globalData.fingerprint[:], globalData.rpcs.Self(), timestamp)
		}
	}
	if globalData.receptors.IsSet() {
		addrsBinary, errAddr := proto.Marshal(&receptor.Addrs{Address: util.GetBytesFromMultiaddr(globalData.receptors.SelfAddress())})
		if nil == errAddr {
			util.LogInfo(log, util.CoYellow, fmt.Sprintf("-><-send self data to P2P ID:%v address:%v", globalData.receptors.ShortID(), util.PrintMaAddrs(globalData.receptors.SelfAddress())))
			if globalData.dnsPeerOnly == UsePeers { //Make self a  hiden node to avoid been connected
				messagebus.Bus.P2P.Send("peer", globalData.receptors.BinaryID(), addrsBinary, timestamp)
			}
		}
	}
	globalData.rpcs.Expire()
	expirePeer(log)

	//if globalData.treeChanged {
	count := globalData.receptors.Tree().Count()
	if count <= MinTreeExpected {
		exhaustiveConnections(log)
	} else {
		globalData.receptors.BalanceTree()
	}

	globalData.receptors.Change(false)
	//}
}

func expirePeer(log *logger.L) {
	now := time.Now()
	tree := globalData.receptors.Tree()
	nextNode := tree.First()
loop:
	for node := nextNode; nil != node; node = nextNode {

		p := node.Value().(*receptor.Data)
		key := node.Key()

		nextNode = node.Next()

		// skip this node's entry
		if globalData.receptors.ID().String() == p.ID.String() {
			continue loop
		}
		if p.Timestamp.Add(announceExpiry).Before(now) {
			tree.Delete(key)
			globalData.receptors.Change(true)
			util.LogDebug(log, util.CoReset, fmt.Sprintf("expirePeer : ID: %v! Timestamp: %s", p.ID.ShortString(), p.Timestamp.Format(timeFormat)))
			idBinary, errID := p.ID.Marshal()
			if nil == errID {
				messagebus.Bus.P2P.Send("@D", idBinary)
				util.LogInfo(log, util.CoYellow, fmt.Sprintf("--><-- Send @D to P2P  ID: %v", p.ID.ShortString()))
			}
		}

	}
}

func exhaustiveConnections(log *logger.L) {
	tree := globalData.receptors.Tree()
	if nil == globalData.receptors.Self() {
		util.LogWarn(log, util.CoRed, fmt.Sprintf("exhaustiveConnections called to early"))
		return // called to early
	}
	// locate this node in the tree
	count := tree.Count()
	for i := 0; i < count; i++ {
		node := tree.Get(i)
		if nil != node {
			p := node.Value().(*receptor.Data)
			if nil != p && !util.IDEqual(p.ID, globalData.receptors.ID()) {
				idBinary, errID := p.ID.Marshal()
				pbAddr := util.GetBytesFromMultiaddr(p.Listeners)
				pbAddrBinary, errMarshal := proto.Marshal(&receptor.Addrs{Address: pbAddr})
				if nil == errID && nil == errMarshal {
					messagebus.Bus.P2P.Send("ES", idBinary, pbAddrBinary)
					util.LogDebug(log, util.CoYellow, fmt.Sprintf("--><-- exhaustiveConnections send to P2P %v : %s  address: %x ", "ES", p.ID.ShortString(), receptor.AddrToString(pbAddrBinary)))
				}
			}
		}
	}
	// locate this node in the tree
}
