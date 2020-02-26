// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/rpc/owner"
)

// OwnedData - data for an ownership request
type OwnedData struct {
	Owner *account.Account
	Start uint64
	Count int
}

// GetOwned - obtain list of owned bitmarks
func (client *Client) GetOwned(ownedConfig *OwnedData) (*owner.OwnerBitmarksReply, error) {

	ownedArgs := owner.OwnerBitmarksArguments{
		Owner: ownedConfig.Owner,
		Start: ownedConfig.Start,
		Count: ownedConfig.Count,
	}

	client.printJson("Owned Request", ownedArgs)

	reply := &owner.OwnerBitmarksReply{}
	err := client.client.Call("Owner.Bitmarks", ownedArgs, reply)
	if nil != err {
		return nil, err
	}

	client.printJson("Owned Reply", reply)

	return reply, nil
}
