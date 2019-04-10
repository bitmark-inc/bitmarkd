// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"time"

	"github.com/bitmark-inc/bitmarkd/block"

	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/peer"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/logger"
)

type Node struct {
	log     *logger.L
	limiter *rate.Limiter
	start   time.Time
	version string
}

// limit for count
const maximumNodeList = 100

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

	if err := rateLimitN(node.limiter, arguments.Count, maximumNodeList); nil != err {
		return err
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

type BlockInfo struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

type InfoReply struct {
	Chain               string    `json:"chain"`
	Mode                string    `json:"mode"`
	Blocks              BlockInfo `json:"blocks"`
	RPCs                uint64    `json:"rpcs"`
	Peers               uint64    `json:"peers"`
	TransactionCounters Counters  `json:"transactionCounters"`
	Difficulty          float64   `json:"difficulty"`
	Version             string    `json:"version"`
	Uptime              string    `json:"uptime"`
	PublicKey           string    `json:"publicKey"`
}

type Counters struct {
	Pending  int `json:"pending"`
	Verified int `json:"verified"`
}

func (node *Node) Info(arguments *InfoArguments, reply *InfoReply) error {

	if err := rateLimit(node.limiter); nil != err {
		return err
	}

	incoming, outgoing := peer.GetCounts()
	reply.Chain = mode.ChainName()
	reply.Mode = mode.String()
	reply.Blocks = BlockInfo{
		Height: blockheader.Height(),
		Hash:   block.LastBlockHash(),
	}
	reply.RPCs = connectionCount.Uint64()
	reply.Peers = incoming + outgoing
	reply.TransactionCounters.Pending, reply.TransactionCounters.Verified = reservoir.ReadCounters()
	reply.Difficulty = difficulty.Current.Reciprocal()
	reply.Version = node.version
	reply.Uptime = time.Since(node.start).String()
	reply.PublicKey = hex.EncodeToString(peer.PublicKey())
	return nil
}
