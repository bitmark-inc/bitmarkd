// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockheader

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/logger"
)

// globals for header
type blockData struct {
	sync.RWMutex // to allow locking

	log *logger.L

	height            uint64             // this is the current block Height
	previousBlock     blockdigest.Digest // and its digest
	previousVersion   uint16             // plus its version
	previousTimestamp uint64             // plus its timestamp

	// set once during initialise
	initialised bool
}

// global data
var globalData blockData

// setup the current block data
func Initialise() error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	log := logger.New("blockheader")
	globalData.log = log
	log.Info("starting…")

	setGenesis()

	log.Infof("block height: %d", globalData.height)
	log.Infof("previous block: %v", globalData.previousBlock)
	log.Infof("previous version: %d", globalData.previousVersion)

	// all data initialised
	globalData.initialised = true

	return nil
}

// shutdown the block header system
func Finalise() error {

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

// reset the block data
func SetGenesis() {
	globalData.Lock()
	setGenesis()
	globalData.Unlock()
}

// internal: must hold lock
func setGenesis() {
	globalData.height = genesis.BlockNumber
	globalData.previousBlock = genesis.LiveGenesisDigest
	globalData.previousVersion = 1
	globalData.previousTimestamp = 0
	if mode.IsTesting() {
		globalData.previousBlock = genesis.TestGenesisDigest
	}
}

// set all data
func Set(height uint64, digest blockdigest.Digest, version uint16, timestamp uint64) {

	globalData.Lock()

	globalData.height = height
	globalData.previousBlock = digest
	globalData.previousVersion = version
	globalData.previousTimestamp = timestamp

	globalData.Unlock()
}

// return all header data
func Get() (uint64, blockdigest.Digest, uint16, uint64) {

	globalData.Lock()
	defer globalData.Unlock()

	return globalData.height, globalData.previousBlock, globalData.previousVersion, globalData.previousTimestamp
}

// get block data for initialising a new block
// returns: previous block digest and the number for the new block
func GetNew() (blockdigest.Digest, uint64) {
	globalData.Lock()
	defer globalData.Unlock()
	nextBlockNumber := globalData.height + 1
	return globalData.previousBlock, nextBlockNumber
}

// current height
func Height() uint64 {

	globalData.Lock()
	defer globalData.Unlock()

	return globalData.height
}
