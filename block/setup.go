// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/genesis"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/logger"
	"sync"
)

// globals for background proccess
type blockData struct {
	sync.RWMutex // to allow locking

	// logger
	log *logger.L

	blockNumber   uint64
	previousBlock blockdigest.Digest

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

	globalData.log = logger.New("block")
	if nil == globalData.log {
		return fault.ErrInvalidLoggerChannel
	}
	globalData.log.Info("starting…")

	// set initial data
	globalData.blockNumber = genesis.BlockNumber + 1
	globalData.previousBlock = genesis.LiveGenesisDigest

	if mode.IsTesting() {
		globalData.previousBlock = genesis.TestGenesisDigest
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

	return nil
}

// get block data
func Get() (blockdigest.Digest, uint64) {
	globalData.Lock()
	defer globalData.Unlock()
	return globalData.previousBlock, globalData.blockNumber
}

// set block data
func Set(header *blockrecord.Header) {
	globalData.Lock()
	defer globalData.Unlock()

	globalData.previousBlock = header.PreviousBlock
	globalData.blockNumber = header.Number
}
