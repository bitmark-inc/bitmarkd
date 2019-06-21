// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockheader

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
)

// DigestForBlock - return the digest for a specific block number
func DigestForBlock(number uint64) (blockdigest.Digest, error) {
	globalData.Lock()
	defer globalData.Unlock()

	// valid block number
	if number <= genesis.BlockNumber {
		if mode.IsTesting() {
			return genesis.TestGenesisDigest, nil
		}
		return genesis.LiveGenesisDigest, nil
	}

	// fetch block and compute digest
	n := make([]byte, 8)
	binary.BigEndian.PutUint64(n, number)
	packed := storage.Pool.Blocks.Get(n) // ***** FIX THIS: possible optimisation is to store the block hashes in a separate index
	if nil == packed {
		return blockdigest.Digest{}, fault.ErrBlockNotFound
	}

	_, digest, _, err := blockrecord.ExtractHeader(packed, 0)

	return digest, err
}
