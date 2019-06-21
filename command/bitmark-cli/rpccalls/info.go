// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/rpc"
)

// GetBitmarkInfo - request status from bitmarkd
func (client *Client) GetBitmarkInfo() (*rpc.InfoReply, error) {
	var reply rpc.InfoReply
	if err := client.client.Call("Node.Info", rpc.InfoArguments{}, &reply); err != nil {
		return nil, err
	}

	return &reply, nil
}
