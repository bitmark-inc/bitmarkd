// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/rpc"
)

// GetBitmarkInfo - request status from bitmarkd (must be matching version)
func (client *Client) GetBitmarkInfo() (*rpc.InfoReply, error) {
	var reply rpc.InfoReply
	if err := client.client.Call("Node.Info", rpc.InfoArguments{}, &reply); err != nil {
		return nil, err
	}

	return &reply, nil
}

// GetBitmarkInfoCompat - request status from bitmarkd (any version)
func (client *Client) GetBitmarkInfoCompat() (map[string]interface{}, error) {
	var reply map[string]interface{}
	if err := client.client.Call("Node.Info", rpc.InfoArguments{}, &reply); err != nil {
		return nil, err
	}

	return reply, nil
}
