// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mode

import (
	"sync"

	"github.com/bitmark-inc/bitmarkd/chain"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// type to hold the mode
type Mode int

// all possible modes
const (
	Stopped Mode = iota
	Resynchronise
	Normal
	maximum
)

var globalData struct {
	sync.RWMutex
	log     *logger.L
	mode    Mode
	testing bool
	chain   string

	// set once during initialise
	initialised bool
}

// set up the mode system
func Initialise(chainName string) error {

	// ensure start up in resynchronise mode
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	globalData.log = logger.New("mode")
	globalData.log.Info("starting…")

	// default settings
	globalData.chain = chainName
	globalData.testing = false
	globalData.mode = Resynchronise

	// override for specific chain
	switch chainName {
	case chain.Bitmark:
		// no change
	case chain.Testing, chain.Local:
		globalData.testing = true
	default:
		globalData.log.Criticalf("mode cannot handle chain: '%s'", chainName)
		return fault.ErrInvalidChain
	}

	// all data initialised
	globalData.initialised = true

	return nil
}

// shutdown mode handling
func Finalise() error {

	if !globalData.initialised {
		return fault.ErrNotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	Set(Stopped)

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// change mode
func Set(mode Mode) {

	if mode >= Stopped && mode < maximum {
		globalData.Lock()
		globalData.mode = mode
		globalData.Unlock()

		globalData.log.Infof("set: %s", mode)
	} else {
		globalData.log.Errorf("ignore invalid set: %d", mode)
	}
}

// detect mode
func Is(mode Mode) bool {
	globalData.RLock()
	defer globalData.RUnlock()
	return mode == globalData.mode
}

// detect mode
func IsNot(mode Mode) bool {
	globalData.RLock()
	defer globalData.RUnlock()
	return mode != globalData.mode
}

// special for testing
func IsTesting() bool {
	globalData.RLock()
	defer globalData.RUnlock()
	return globalData.testing
}

// name of the current chain
func ChainName() string {
	globalData.RLock()
	defer globalData.RUnlock()
	return globalData.chain
}

// current mode represented as a string
func String() string {
	globalData.RLock()
	defer globalData.RUnlock()
	return globalData.mode.String()
}

// current mode rep[resented as a string
func (m Mode) String() string {
	switch m {
	case Stopped:
		return "Stopped"
	case Resynchronise:
		return "Resynchronise"
	case Normal:
		return "Normal"
	default:
		return "*Unknown*"
	}
}
