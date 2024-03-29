// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/rpc/share"
)

// BalanceData - the parameters for a balance request
type BalanceData struct {
	Owner   *account.Account
	ShareId string
	Count   int
}

// GetBalance - retrieve some balance data
func (client *Client) GetBalance(balanceConfig *BalanceData) (*share.BalanceReply, error) {

	var shareId merkle.Digest // if not present the all zero id
	if balanceConfig.ShareId != "" {
		if err := shareId.UnmarshalText([]byte(balanceConfig.ShareId)); err != nil {
			return nil, err
		}
	}

	balanceArgs := share.BalanceArguments{
		Owner:   balanceConfig.Owner,
		ShareId: shareId,
		Count:   balanceConfig.Count,
	}

	client.printJson("Balance Request", balanceArgs)

	reply := &share.BalanceReply{}
	err := client.client.Call("Share.Balance", balanceArgs, reply)
	if err != nil {
		return nil, err
	}

	client.printJson("Balance Reply", reply)

	return reply, nil
}
