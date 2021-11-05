// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/rpc/node"
)

// BlockDumpData - the parameters for a blockDump request
type BlockDumpData struct {
	Block uint64
	Count int
	Txs   bool
}

// GetBlocks - retrieve some blocks
func (client *Client) GetBlocks(blockDumpConfig *BlockDumpData) (*node.BlockDumpRangeReply, error) {

	blockDumpArgs := node.BlockDumpRangeArguments{
		Height: blockDumpConfig.Block,
		Count:  blockDumpConfig.Count,
		Txs:    blockDumpConfig.Txs,
	}

	client.printJson("BlockDump Request", blockDumpArgs)

	reply := &node.BlockDumpRangeReply{}
	err := client.client.Call("Node.BlockDumpRange", blockDumpArgs, reply)
	if nil != err {
		return nil, err
	}

	client.printJson("BlockDump Reply", reply)

	return reply, nil
}
