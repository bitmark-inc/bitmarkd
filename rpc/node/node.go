// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package node

import (
	"time"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/bitmark-inc/bitmarkd/counter"

	"github.com/bitmark-inc/bitmarkd/rpc/ratelimit"

	"github.com/bitmark-inc/bitmarkd/storage"

	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/announce/rpc"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/difficulty"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/p2p"
	"github.com/bitmark-inc/bitmarkd/proof"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/logger"
)

const (
	rateLimitNode = 200
	rateBurstNode = 100
)

// Node - type for RPC calls
type Node struct {
	Log      *logger.L
	Limiter  *rate.Limiter
	Start    time.Time
	Version  string
	Announce announce.Announce
	Pool     storage.Handle
	counter  *counter.Counter
}

// limit for count
const maximumNodeList = 100

// ---

// NodeArguments - arguments for RPC
type NodeArguments struct {
	Start uint64 `json:"Start,string"`
	Count int    `json:"count"`
}

// NodeReply - result from RPC
type NodeReply struct {
	Nodes     []rpc.Entry `json:"nodes"`
	NextStart uint64      `json:"nextStart,string"`
}

func New(log *logger.L, pools reservoir.Handles, start time.Time, version string, counter *counter.Counter, ann announce.Announce) *Node {
	return &Node{
		Log:      log,
		Limiter:  rate.NewLimiter(rateLimitNode, rateBurstNode),
		Start:    start,
		Version:  version,
		Announce: ann,
		Pool:     pools.Blocks,
		counter:  counter,
	}
}

// List - list all node offering RPC functionality
func (node *Node) List(arguments *NodeArguments, reply *NodeReply) error {

	if err := ratelimit.LimitN(node.Limiter, arguments.Count, maximumNodeList); nil != err {
		return err
	}

	nodes, nextStart, err := node.Announce.Fetch(arguments.Start, arguments.Count)
	if nil != err {
		return err
	}
	reply.Nodes = nodes
	reply.NextStart = nextStart

	return nil
}

// ---

// InfoArguments - empty arguments for info request
type InfoArguments struct{}

// InfoReply - results from info request
type InfoReply struct {
	Chain               string    `json:"chain"`
	Mode                string    `json:"mode"`
	Block               BlockInfo `json:"block"`
	Miner               MinerInfo `json:"miner"`
	RPCs                uint64    `json:"rpcs"`
	Peers               uint64    `json:"peers"`
	TransactionCounters Counters  `json:"transactionCounters"`
	Difficulty          float64   `json:"difficulty"`
	Hashrate            float64   `json:"hashrate"`
	Version             string    `json:"Version"`
	Uptime              string    `json:"uptime"`
	PublicKey           string    `json:"publicKey"`
}

// BlockInfo - the highest block held by the node
type BlockInfo struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

// Counters - transaction counters
type Counters struct {
	Pending  int `json:"pending"`
	Verified int `json:"verified"`
}

// MinerInfo - miner info, include success / failed mined block count
type MinerInfo struct {
	Success uint64 `json:"success"`
	Failed  uint64 `json:"failed"`
}

// Info - return some information about this node
// only enough for clients to determine node state
// for more detail information use HTTP GET requests
func (node *Node) Info(_ *InfoArguments, reply *InfoReply) error {

	if err := ratelimit.Limit(node.Limiter); nil != err {
		return err
	}

	if node.Pool == nil {
		return fault.DatabaseIsNotSet
	}

	connCounts := uint64(p2p.GetNetworkMetricConnCount())

	reply.Chain = mode.ChainName()
	reply.Mode = mode.String()
	reply.Block = BlockInfo{
		Height: blockheader.Height(),
		Hash:   block.LastBlockHash(node.Pool),
	}
	reply.Miner = MinerInfo{
		Success: uint64(proof.MinedBlocks()),
		Failed:  uint64(proof.FailMinedBlocks()),
	}
	reply.RPCs = node.counter.Uint64()
	reply.Peers = connCounts
	reply.TransactionCounters.Pending, reply.TransactionCounters.Verified = reservoir.ReadCounters()
	reply.Difficulty = difficulty.Current.Value()
	reply.Hashrate = difficulty.Hashrate()
	reply.Version = node.Version
	reply.Uptime = time.Since(node.Start).String()
	//TODO: make it ID not public key, this is base58Encoded
	reply.PublicKey = p2p.ID().Pretty()
	return nil
}
