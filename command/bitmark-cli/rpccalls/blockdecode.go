// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/rpc/node"
)

// BlockDecodeData - the parameters for a blockDecode request
type BlockDecodeData struct {
	Packed []byte
}

// DecodeBlock - retrieve some blocks
func (client *Client) DecodeBlock(blockDecodeConfig *BlockDecodeData) (*node.BlockDecodeReply, error) {

	blockDecodeArgs := node.BlockDecodeArguments{
		Packed: blockDecodeConfig.Packed,
	}

	client.printJson("BlockDecode Request", blockDecodeArgs)

	reply := &node.BlockDecodeReply{}
	err := client.client.Call("Node.BlockDecode", blockDecodeArgs, reply)
	if err != nil {
		return nil, err
	}

	client.printJson("BlockDecode Reply", reply)

	return reply, nil
}
