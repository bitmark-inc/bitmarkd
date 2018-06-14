// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/logger"
)

type Node struct {
	log     *logger.L
	start   time.Time
	version string
}

// list the RPC services available in the network

type NodeArguments struct {
	Start uint64 `json:"start,string"`
	Count int    `json:"count"`
}

type NodeReply struct {
	Nodes     []announce.RPCEntry `json:"nodes"`
	NextStart uint64              `json:"nextStart,string"`
}

func (node *Node) List(arguments *NodeArguments, reply *NodeReply) error {
	if arguments.Count <= 0 || arguments.Count > 100 {
		return fault.ErrInvalidCount
	}
	nodes, nextStart, err := announce.FetchRPCs(arguments.Start, arguments.Count)
	if nil != err {
		return err
	}
	reply.Nodes = nodes
	reply.NextStart = nextStart

	return nil
}

// return some information about this node
// only enough for clients to determin node state
// for more detaile information use http GET requests

type InfoArguments struct{}

type InfoReply struct {
	Chain               string   `json:"chain"`
	Mode                string   `json:"mode"`
	Blocks              uint64   `json:"blocks"`
	RPCs                uint64   `json:"rpcs"`
	Peers               uint64   `json:"peers"`
	TransactionCounters Counters `json:"transactionCounters"`
	Difficulty          float64  `json:"difficulty"`
	Version             string   `json:"version"`
	Uptime              string   `json:"uptime"`
	PublicKey           string   `json:"publicKey"`
}

type Counters struct {
	Pending  int `json:"pending"`
	Verified int `json:"verified"`
}

func (node *Node) Info(arguments *InfoArguments, reply *InfoReply) error {

	l, r := peer.GetCounts()
	reply.Chain = mode.ChainName()
	reply.Mode = mode.String()
	reply.Blocks = block.GetHeight()
	reply.RPCs = connectionCount.Uint64()
	reply.Peers = l + r
	reply.TransactionCounters.Pending, reply.TransactionCounters.Verified = reservoir.ReadCounters()
	reply.Difficulty = difficulty.Current.Reciprocal()
	reply.Version = node.version
	reply.Uptime = time.Since(node.start).String()
	reply.PublicKey = hex.EncodeToString(peer.PublicKey())
	return nil
}
