// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/storage"
)

func doBlockHeaderHash() error {
	return storage.Pool.Blocks.NewFetchCursor().Map(recoverBlockHeaderHash)
}

func recoverBlockHeaderHash(blockNumberBytes []byte, packedBlock []byte) error {
	globalData.Lock()
	defer globalData.Unlock()

	// TODO: decide if we need to disable reservoir when migrating the block db
	// reservoir.Disable()
	// defer reservoir.Enable()

	// reservoir.ClearSpend()
	trx, err := storage.NewDBTransaction()
	if nil != err {
		return err
	}

	blockNumber := binary.BigEndian.Uint64(blockNumberBytes)

	blockHeaderHashBytes := trx.Get(storage.Pool.BlockHeaderHash, blockNumberBytes)
	if blockHeaderHashBytes == nil {
		digest, err := blockrecord.ComputeHeaderHash(packedBlock)
		if nil != err {
			return err
		}

		trx.Put(storage.Pool.BlockHeaderHash, blockNumberBytes, digest[:], []byte{})
	}
	trx.Commit()

	globalData.log.Debugf("rebuilt block: %d", blockNumber)

	return nil
}
