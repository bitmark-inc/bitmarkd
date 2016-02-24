// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bilateralrpc"
	"github.com/bitmark-inc/logger"
	"sort"
)

// type for returned block numbers
type BlockNumberResult struct {
	From  string
	Reply BlockNumberReply
	Err   error
}

// ByBlockNumber implements sort.Interface for []BlockNumberReply based on
// the Reply.Number field.
type ByBlockNumber []BlockNumberResult

// sort interface
// Note: need '>' to get highest block first
func (a ByBlockNumber) Len() int           { return len(a) }
func (a ByBlockNumber) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByBlockNumber) Less(i, j int) bool { return a[i].Reply.Number > a[j].Reply.Number }

// get the highest block number from peers
// return it and the peers name
func highestBlockNumber(server *bilateralrpc.Bilateral, log *logger.L) (uint64, string, bool) {

	args := BlockNumberArguments{}
	var result []BlockNumberResult
	if err := server.Call(bilateralrpc.SendToAll, "Block.Number", args, &result, 0); nil != err {
		log.Errorf("highestBlockNumber: err = %v", err)
		return 0, "", false
	}

	l := len(result)
	if 0 == l {
		log.Infof("highestBlockNumber: no results")
		return 0, "", false
	}

	log.Infof("highestBlockNumber: unsorted results: %v", result)

	// received some values
	sort.Sort(ByBlockNumber(result))

	log.Infof("highestBlockNumber: sorted results: %v", result)

	for _, v := range result {
		if nil == v.Err {
			highest := v.Reply.Number
			from := v.From
			return highest, from, true
		}
	}

	// all peers returned an error
	return 0, "", false
}
