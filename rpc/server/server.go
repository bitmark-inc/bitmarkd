// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package server

import (
	"net/rpc"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/assets"
	"github.com/bitmark-inc/bitmarkd/rpc/bitmark"
	"github.com/bitmark-inc/bitmarkd/rpc/bitmarks"
	"github.com/bitmark-inc/bitmarkd/rpc/blockowner"
	"github.com/bitmark-inc/bitmarkd/rpc/node"
	"github.com/bitmark-inc/bitmarkd/rpc/owner"
	"github.com/bitmark-inc/bitmarkd/rpc/share"
	"github.com/bitmark-inc/bitmarkd/rpc/transaction"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/logger"
)

func Create(log *logger.L, version string, rpcCount *counter.Counter) *rpc.Server {

	start := time.Now().UTC()
	pools := reservoir.Handles{
		Assets:            storage.Pool.Assets,
		BlockOwnerPayment: storage.Pool.BlockOwnerPayment,
		Blocks:            storage.Pool.Blocks,
		Transactions:      storage.Pool.Transactions,
		OwnerTxIndex:      storage.Pool.OwnerTxIndex,
		OwnerData:         storage.Pool.OwnerData,
		Share:             storage.Pool.Shares,
		ShareQuantity:     storage.Pool.ShareQuantity,
	}

	server := rpc.NewServer()

	_ = server.Register(assets.New(log, pools, mode.Is, mode.IsTesting))
	_ = server.Register(bitmark.New(log, pools, mode.Is, mode.IsTesting, reservoir.Get()))
	_ = server.Register(bitmarks.New(log, pools, mode.Is, reservoir.Get()))
	_ = server.Register(owner.New(log, pools, ownership.Get()))
	_ = server.Register(node.New(log, pools, start, version, rpcCount, announce.Get()))
	_ = server.Register(transaction.New(log, start, reservoir.Get()))
	_ = server.Register(blockowner.New(log, pools, mode.Is, mode.IsTesting, reservoir.Get(), blockrecord.Get()))
	_ = server.Register(share.New(log, mode.Is, reservoir.Get()))

	return server
}
