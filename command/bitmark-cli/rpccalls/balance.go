// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/rpc"
)

// BalanceData - the parameters for a balance request
type BalanceData struct {
	Owner   *keypair.KeyPair
	ShareId string
	Count   int
}

// GetBalance - retrieve some balance data
func (client *Client) GetBalance(balanceConfig *BalanceData) (*rpc.ShareBalanceReply, error) {

	var shareId merkle.Digest // if not present the all zero id
	if "" != balanceConfig.ShareId {
		if err := shareId.UnmarshalText([]byte(balanceConfig.ShareId)); nil != err {
			return nil, err
		}
	}

	ownerAddress := makeAddress(balanceConfig.Owner, client.testnet)
	balanceArgs := rpc.ShareBalanceArguments{
		Owner:   ownerAddress,
		ShareId: shareId,
		Count:   balanceConfig.Count,
	}

	client.printJson("Balance Request", balanceArgs)

	reply := &rpc.ShareBalanceReply{}
	err := client.client.Call("Share.Balance", balanceArgs, reply)
	if nil != err {
		return nil, err
	}

	client.printJson("Balance Reply", reply)

	return reply, nil
}
