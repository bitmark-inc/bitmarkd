// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockring

import (
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/logger"
	"sync"
)

// internal constants
const (
	Size = 20 // size of ring buffer
)

// type to hold a block's digest and its crc64 check code
type ringBuffer struct {
	number uint64             // block number
	crc    uint64             // CRC64_ECMA(block_number, complete_block_bytes)
	digest blockdigest.Digest // header digest
}

// globals for background proccess
type ringData struct {
	sync.RWMutex // to allow locking

	log *logger.L

	height uint64

	ring      [Size]ringBuffer
	ringIndex int

	// set once during initialise
	initialised bool
}

// global data
var globalData ringData

// setup the current block data
func Initialise() error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	log := logger.New("ring")
	globalData.log = log
	log.Info("starting…")

	// zero height and fill ring with default values
	if err := clearRingBuffer(log); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	return nil
}

// shudown the block system
func Finalise() error {
	globalData.Lock()
	defer globalData.Unlock()

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
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

// fetch a digest from the ring if present
func DigestForBlock(number uint64) *blockdigest.Digest {
	globalData.Lock()
	defer globalData.Unlock()

	// check if in the cache
	i := globalData.height - number
	if i < Size {
		j := globalData.ringIndex - 1 - int(i)
		if j < 0 {
			j += Size
		}
		if number != globalData.ring[j].number {
			logger.Panicf("block.DigestForBlock: ring buffer corrupted block number, actual: %d  expected: %d", globalData.ring[j].number, number)
		}
		return &globalData.ring[j].digest
	}
	return nil
}

// store a block ant its digest
func Put(number uint64, digest blockdigest.Digest, packedBlock []byte) {

	// start of critical section
	globalData.Lock()
	defer globalData.Unlock()

	globalData.log.Infof("put block number: %d", number)

	if 0 != globalData.height && globalData.height+1 != number {
		logger.Panicf("block number: actual: %d  expected: %d", number, globalData.height+1)
	}

	i := globalData.ringIndex
	globalData.ring[i].number = number
	globalData.ring[i].digest = digest
	globalData.ring[i].crc = CRC(number, packedBlock)
	i = i + 1
	if i >= len(globalData.ring) {
		i = 0
	}
	globalData.ringIndex = i

	globalData.height = number
}

func Clear(log *logger.L) error {
	globalData.Lock()
	defer globalData.Unlock()
	return clearRingBuffer(log)
}

// must hold lock to call this
func clearRingBuffer(log *logger.L) error {

	// set initial crc depending on mode
	number := genesis.BlockNumber
	digest := genesis.LiveGenesisDigest
	block := genesis.LiveGenesisBlock
	if mode.IsTesting() {
		digest = genesis.TestGenesisDigest
		block = genesis.TestGenesisBlock
	}

	// default CRC of appropriate genesis block
	crc := CRC(number, block)

	// fill ring with default values
	globalData.ringIndex = 0
	for i := 0; i < len(globalData.ring); i += 1 {
		globalData.ring[i].number = number
		globalData.ring[i].digest = digest
		globalData.ring[i].crc = crc
	}

	// zero the height so next put will succeed
	globalData.height = 0

	return nil
}
