// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// get block data for initialising a new block
// returns: previous block digest and the number for the new block
func Get() (blockdigest.Digest, uint64) {
	globalData.Lock()
	defer globalData.Unlock()
	nextBlockNumber := globalData.height + 1
	return globalData.previousBlock, nextBlockNumber
}

// get the current height
func GetHeight() uint64 {
	globalData.Lock()
	height := globalData.height
	globalData.Unlock()
	return height
}

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

	// check if in the cache
	if number > genesis.BlockNumber && number <= globalData.height {
		i := globalData.height - number
		if i < ringSize {
			j := globalData.ringIndex - 1 - int(i)
			if j < 0 {
				j = ringSize - 1
			}
			if number != globalData.ring[j].number {
				fault.Panicf("block.DigestForBlock: ring buffer corrupted block number, actual: %d  expected: %d", globalData.ring[j].number, number)
			}
			return globalData.ring[j].digest, nil
		}
	}

	// no cache, fetch block and compute digest
	n := make([]byte, 8)
	binary.BigEndian.PutUint64(n, number)
	packed := storage.Pool.Blocks.Get(n) // ***** FIX THIS: possible optimisation is to store the block hashes in a separate index
	if nil == packed {
		return blockdigest.Digest{}, fault.ErrBlockNotFound
	}
	return blockrecord.PackedHeader(packed).Digest(), nil
}

// fetch latest crc value
func GetLatestCRC() uint64 {
	globalData.Lock()
	i := globalData.ringIndex - 1
	if i < 0 {
		i = len(globalData.ring) - 1
	}
	crc := globalData.ring[i].crc
	globalData.Unlock()
	return crc
}

// // number of blocks to consider for
// const topN = 50

// // get a payment record from a random block
// func GetRandomPayment() *transactionrecord.Payment {
// 	high := GetHeight()
// 	low := 2
// 	if high < low {
// 		return
// 	}
// 	if high > topN+1 {
// 		low = high - topN
// 	}
// 	// note: low >= 2
// 	if high == low {
// 		return GetPayment(h)
// 	}
//	const denominator = 256 // 65536 // depends on random bytes
// 	random=[0..denominator-1] // or need bigger range
// 	n:= random *(high - low)/denominator + low // uniform [low..high)
// 	return GetPayment(n)
// }

// // get a payment record from a specific block
// func GetPaymentNumbered(blockNumber uint64) *transactionrecord.Payment {
// 	// get block number of issue
// 	bKey := make([]byte, 8)
// 	binary.BigEndian.PutUint64(bKey, blockNumber)
// 	return GetPayment(bKey)
// }

// get a payment record from a specific block given the blocks 8 byte big endian key
func GetPayment(blockNumberKey []byte) *transactionrecord.Payment {

	if 8 != len(blockNumberKey) {
		fault.Panicf("block.GetPayment: block number need 8 bytes: %x", blockNumberKey)
	}

	blockOwnerData := storage.Pool.BlockOwners.Get(blockNumberKey)
	if nil == blockOwnerData {
		fault.Panicf("block.GetPayment: no block owner data for block number: %x", blockNumberKey)
	}

	c, err := currency.FromUint64(binary.BigEndian.Uint64(blockOwnerData[:8]))
	if nil != err {
		fault.Panicf("block.GetPayment: block currency invalid error: %v", err)
	}
	return &transactionrecord.Payment{
		Currency: c,
		Address:  string(blockOwnerData[8:]),
		Amount:   5000, // ***** FIX THIS: what is the correct value for issuer?
	}
}
