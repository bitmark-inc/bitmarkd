// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/bitmarkd/rpc"
)

// OwnedData - data for an ownership request
type OwnedData struct {
	Owner *keypair.KeyPair
	Start uint64
	Count int
}

// GetOwned - obtain list of owned bitmarks
func (client *Client) GetOwned(ownedConfig *OwnedData) (*rpc.OwnerBitmarksReply, error) {

	ownerAddress := makeAddress(ownedConfig.Owner, client.testnet)
	ownedArgs := rpc.OwnerBitmarksArguments{
		Owner: ownerAddress,
		Start: ownedConfig.Start,
		Count: ownedConfig.Count,
	}

	client.printJson("Owned Request", ownedArgs)

	reply := &rpc.OwnerBitmarksReply{}
	err := client.client.Call("Owner.Bitmarks", ownedArgs, reply)
	if nil != err {
		return nil, err
	}

	client.printJson("Owned Reply", reply)

	return reply, nil
}
