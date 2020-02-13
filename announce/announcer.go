// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/parameter"

	"github.com/bitmark-inc/bitmarkd/announce/observer"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	"github.com/golang/protobuf/proto"
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
func (ann *announcer) Run(arg interface{}, shutdown <-chan struct{}) {
	log := ann.log

	log.Info("starting…")

	queue := arg.(<-chan messagebus.Message)

	observers := []observer.Observer{
		observer.NewReconnect(globalData.receptors),
		observer.NewUpdatetime(globalData.receptors, globalData.log),
		observer.NewAddpeer(globalData.receptors, globalData.log),
		observer.NewAddrpc(globalData.rpcs, globalData.log),
		observer.NewSelf(globalData.receptors, globalData.log),
	}

	delay := time.After(parameter.InitialiseInterval)
loop:
	for {
		log.Debug("waiting…")
		select {
		case <-shutdown:
			break loop
		case item := <-queue:
			util.LogInfo(log, util.CoReset, fmt.Sprintf("received control: %s  parameters: %x", item.Command, item.Parameters))

			for _, o := range observers {
				o.Update(item.Command, item.Parameters)
			}

		case <-delay:
			delay = time.After(parameter.InitialiseInterval)
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
	globalData.receptors.Expire()

	//if globalData.treeChanged {
	count := globalData.receptors.Tree().Count()
	if count <= parameter.MinTreeExpected {
		exhaustiveConnections(log)
	} else {
		globalData.receptors.BalanceTree()
	}

	globalData.receptors.Change(false)
	//}
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
