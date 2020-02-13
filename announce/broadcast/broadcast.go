// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/observer"

	"github.com/bitmark-inc/bitmarkd/background"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"

	"github.com/bitmark-inc/bitmarkd/announce/rpc"

	"github.com/bitmark-inc/bitmarkd/announce/parameter"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/logger"
	"github.com/golang/protobuf/proto"
)

type DNSType bool

const (
	DnsOnly  DNSType = true
	UsePeers DNSType = false
)

type broadcast struct {
	sync.RWMutex
	log           *logger.L
	receptors     receptor.Receptor
	rpcs          rpc.RPC
	myFingerprint fingerprint.Type
	dnsType       DNSType
}

func NewBroadcast(log *logger.L, receptors receptor.Receptor, rpcs rpc.RPC, myFingerprint fingerprint.Type, dnsType DNSType) background.Process {
	log.Info("initialising…")
	return &broadcast{
		log:           log,
		receptors:     receptors,
		rpcs:          rpcs,
		myFingerprint: myFingerprint,
		dnsType:       dnsType,
	}
}

// wait for incoming requests, process them and reply
func (b *broadcast) Run(arg interface{}, shutdown <-chan struct{}) {
	log := b.log

	log.Info("starting…")

	queue := arg.(chan messagebus.Message)

	observers := []observer.Observer{
		observer.NewReconnect(b.receptors),
		observer.NewUpdatetime(b.receptors, b.log),
		observer.NewAddpeer(b.receptors, b.log),
		observer.NewAddrpc(b.rpcs, b.log),
		observer.NewSelf(b.receptors, b.log),
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
			b.process()
		}
	}
}

// process the announcement and return response to receptor
func (b *broadcast) process() {
	log := b.log

	util.LogInfo(log, util.CoReset, "process starting…")
	b.Lock()
	defer b.Unlock()

	// get a big endian Timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	// announce this nodes IP and ports to other peers
	if b.rpcs.IsSet() {
		log.Debugf("send rpc: %x", b.myFingerprint)
		if b.dnsType == UsePeers { //Make self a  hiden rpc node to avoid been connected
			messagebus.Bus.P2P.Send("rpc", b.myFingerprint[:], b.rpcs.Self(), timestamp)
		}
	}
	if b.receptors.IsSet() {
		addrsBinary, errAddr := proto.Marshal(&receptor.Addrs{Address: util.GetBytesFromMultiaddr(b.receptors.SelfAddress())})
		if nil == errAddr {
			util.LogInfo(log, util.CoYellow, fmt.Sprintf("-><-send self data to P2P ID:%v address:%v", b.receptors.ShortID(), util.PrintMaAddrs(b.receptors.SelfAddress())))
			if b.dnsType == UsePeers { //Make self a  hiden node to avoid been connected
				messagebus.Bus.P2P.Send("peer", b.receptors.BinaryID(), addrsBinary, timestamp)
			}
		}
	}
	b.rpcs.Expire()
	b.receptors.Expire()

	//if globalData.treeChanged {
	count := b.receptors.Tree().Count()
	if count <= parameter.MinTreeExpected {
		exhaustiveConnections(log, b.receptors)
	} else {
		b.receptors.BalanceTree()
	}

	b.receptors.Change(false)
	//}
}

func exhaustiveConnections(log *logger.L, receptors receptor.Receptor) {
	tree := receptors.Tree()
	if nil == receptors.Self() {
		util.LogWarn(log, util.CoRed, fmt.Sprintf("exhaustiveConnections called to early"))
		return // called to early
	}
	// locate this node in the tree
	count := tree.Count()
	for i := 0; i < count; i++ {
		node := tree.Get(i)
		if nil != node {
			p := node.Value().(*receptor.Data)
			if nil != p && !util.IDEqual(p.ID, receptors.ID()) {
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
