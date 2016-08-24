// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/version"
	"github.com/bitmark-inc/logger"
	"time"
)

type Node struct {
	log   *logger.L
	start time.Time
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

	return err
}

// return some information about this node

type InfoArguments struct{}

type InfoReply struct {
	Chain  string `json:"chain"`
	Mode   string `json:"mode"`
	Blocks uint64 `json:"blocks"`
	// Peers    int     `json:"peers"`
	RPCs uint64 `json:"rpcs"`
	// Miners   uint64  `json:"miners"`
	Difficulty float64 `json:"difficulty"`
	Version    string  `json:"version"`
	Uptime     string  `json:"uptime"`
}

func (node *Node) Info(arguments *InfoArguments, reply *InfoReply) error {

	reply.Chain = mode.ChainName()
	reply.Mode = mode.String()
	reply.Blocks = block.GetHeight()
	// reply.Peers = peer.ConnectionCount()
	reply.RPCs = connectionCount.Uint64()
	// reply.Miners = mine.ConnectionCount()
	reply.Difficulty = difficulty.Current.Reciprocal()
	reply.Version = version.Version
	reply.Uptime = time.Since(node.start).String()

	return nil
}
