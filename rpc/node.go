// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

// import (
// 	"github.com/bitmark-inc/bitmarkd/announce"
// 	"github.com/bitmark-inc/bitmarkd/block"
// 	"github.com/bitmark-inc/bitmarkd/difficulty"
// 	"github.com/bitmark-inc/bitmarkd/gnomon"
// 	"github.com/bitmark-inc/bitmarkd/mine"
// 	"github.com/bitmark-inc/bitmarkd/mode"
// 	"github.com/bitmark-inc/bitmarkd/peer"
// 	"github.com/bitmark-inc/bitmarkd/transaction"
// 	"github.com/bitmark-inc/bitmarkd/version"
// 	"github.com/bitmark-inc/logger"
// 	"strings"
// 	"time"
// )

// // --------------------

// // e.g.
// // {"id":1,"method":"Node.List","params":[{"Start":null,"Count":10}]}
// // {"id":2,"method":"Node.Peers","params":[{"Start":null,"Count":10}]}

// type Node struct {
// 	log   *logger.L
// 	start time.Time
// }

// type NodeArguments struct {
// 	Start *gnomon.Cursor `json:"start"`
// 	Count int            `json:"count"`
// }

// type NodeReply struct {
// 	Addresses []string       `json:"addresses"`
// 	NextStart *gnomon.Cursor `json:"nextStart"`
// }

// // p2p peers for DEBUGGING
// func (node *Node) Peers(arguments *NodeArguments, reply *NodeReply) error {
// 	if arguments.Count <= 0 {
// 		arguments.Count = 10
// 	}
// 	peers, nextStart, err := announce.RecentPeers(arguments.Start, arguments.Count, announce.TypePeer)
// 	if nil == err {
// 		for _, p := range peers {
// 			recent := p.(announce.RecentData)
// 			reply.Addresses = append(reply.Addresses, recent.Address)
// 		}
// 	}
// 	reply.NextStart = nextStart
// 	return err
// }

// func (node *Node) List(arguments *NodeArguments, reply *NodeReply) error {
// 	if arguments.Count <= 0 {
// 		arguments.Count = 10
// 	}
// 	peers, nextStart, err := announce.RecentPeers(arguments.Start, arguments.Count, announce.TypeRPC)
// 	if nil == err {
// 		for _, p := range peers {
// 			recent := p.(announce.RecentData)
// 			// quick hack to exclude localhost/IPv6 - client not ready
// 			if strings.HasPrefix(recent.Address, "[") || strings.HasPrefix(recent.Address, "127.") {
// 				continue
// 			}
// 			reply.Addresses = append(reply.Addresses, recent.Address)
// 		}
// 	}
// 	reply.NextStart = nextStart
// 	return err
// }

// // return some information about this node
// // ---------------------------------------

// type InfoArguments struct{}

// type InfoReply struct {
// 	Chain    string  `json:"chain"`
// 	Mode     string  `json:"mode"`
// 	Blocks   uint64  `json:"blocks"`
// 	Peers    int     `json:"peers"`
// 	RPCs     uint64  `json:"rpcs"`
// 	Miners   uint64  `json:"miners"`
// 	Pdiff    float64 `json:"pdiff"`
// 	Pending  uint64  `json:"pending"`
// 	Verified uint64  `json:"verified"`
// 	Version  string  `json:"version"`
// 	Uptime   string  `json:"uptime"`
// }

// func (node *Node) Info(arguments *InfoArguments, reply *InfoReply) error {

// 	reply.Chain = mode.ChainName()
// 	reply.Mode = mode.String()
// 	reply.Blocks = block.Number() - 1
// 	reply.Peers = peer.ConnectionCount()
// 	reply.RPCs = connectionCount.Uint64()
// 	reply.Miners = mine.ConnectionCount()
// 	reply.Pdiff = difficulty.Current.Pdiff()
// 	reply.Version = version.Version
// 	reply.Uptime = time.Since(node.start).String()
// 	transaction.ReadCounters(&reply.Pending, &reply.Verified)

// 	return nil
// }
