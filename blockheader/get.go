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

const (
	cacheSize = 10
)

type cachedBlockDigest struct {
	blockNumber uint64
	digest      blockdigest.Digest
}

var cached [cacheSize]cachedBlockDigest
var cacheIndex int

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

	digest := digestFromCache(number)
	if !digest.IsEmpty() {
		return digest, nil
	}

	// fetch block and compute digest
	n := make([]byte, 8)
	binary.BigEndian.PutUint64(n, number)

	digest = blockrecord.DigestFromHashPool(storage.Pool.BlockHeaderHash, n)
	if !digest.IsEmpty() {
		addToCache(number, digest)
		return digest, nil
	}

	digest, err := genDigestFromPool(storage.Pool.Blocks, n)
	if nil != err {
		return blockdigest.Digest{}, err
	}

	addToCache(number, digest)
	return digest, err
}

func ClearCache() {
	cached = *new([cacheSize]cachedBlockDigest)
}

func digestFromCache(blockNumber uint64) blockdigest.Digest {
	for _, c := range cached {
		if c.blockNumber == blockNumber {
			return c.digest
		}
	}
	return blockdigest.Digest{}
}

func addToCache(blockNumber uint64, digest blockdigest.Digest) {
	cached[cacheIndex] = cachedBlockDigest{
		blockNumber: blockNumber,
		digest:      digest,
	}
	incrementCacheIndex()
}

func incrementCacheIndex() {
	if cacheSize-1 == cacheIndex {
		cacheIndex = 0
	} else {
		cacheIndex++
	}
}

func genDigestFromPool(pool storage.Handle, blockNumber []byte) (blockdigest.Digest, error) {
	packed := pool.Get(blockNumber)
	if nil == packed {
		return blockdigest.Digest{}, fault.BlockNotFound
	}

	_, digest, _, err := blockrecord.ExtractHeader(packed, 0, false)

	return digest, err
}
