// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

type Asset struct {
	log *logger.L
}

// ------------------------------------------------------------

type AssetGetArguments struct {
	AssetIndex transaction.AssetIndex
}

type AssetGetReply struct {
	Digest     transaction.Link
	AssetIndex transaction.AssetIndex
	Data       []byte
}

// read a specific asset
func (t *Asset) Get(arguments *AssetGetArguments, reply *AssetGetReply) error {
	_, txid, found := arguments.AssetIndex.Read()
	if !found {
		return fault.ErrAssetNotFound
	}

	_, data, found := txid.Read()
	if !found {
		return fault.ErrAssetNotFound
	}

	reply.Digest = txid
	reply.AssetIndex = arguments.AssetIndex
	reply.Data = data
	return nil
}
