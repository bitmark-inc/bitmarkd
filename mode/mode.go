// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package mode

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	"sync"
)

// type to hold the mode
type Mode int

// all possible modes
const (
	Stopped       = Mode(iota)
	Resynchronise = Mode(iota)
	Normal        = Mode(iota)
	maximum       = Mode(iota)
)

var globals struct {
	sync.RWMutex
	log  *logger.L
	mode Mode

	// for enabling test mode
	testing bool
	once    bool // to ensure that repeated swapping is disallowed
}

// set up the mode system
func Initialise() {

	// ensure strt up in resynchronise mode
	globals.Lock()
	globals.log = logger.New("mode")
	globals.mode = Resynchronise
	globals.once = false
	globals.testing = false
	globals.Unlock()

	globals.log.Info("starting…")
}

// shutdown mode handling
func Finalise() {
	Set(Stopped)
	globals.log.Info("shutting down…")
}

// change mode
func Set(mode Mode) {

	if mode >= Stopped && mode < maximum {
		globals.Lock()
		globals.mode = mode
		globals.Unlock()

		globals.log.Infof("set: %d", mode)
	} else {
		globals.log.Errorf("ignore invalid set: %d", mode)
	}
}

// detect mode
func Is(mode Mode) bool {
	globals.RLock()
	defer globals.RUnlock()
	return mode == globals.mode
}

// detect mode
func IsNot(mode Mode) bool {
	globals.RLock()
	defer globals.RUnlock()
	return mode != globals.mode
}

// for testing
func SetTesting(testing bool) {
	globals.Lock()
	defer globals.Unlock()

	// no change is ok
	if testing == globals.testing {
		return
	}

	// only allow one change
	if globals.once {
		if nil != globals.log {
			globals.log.Critical("cannot change testing mode a second time")
		}
		fault.Panic("cannot change testing mode a second time")
	}

	globals.testing = testing
	globals.once = true
}

// special for testing
func IsTesting() bool {
	globals.RLock()
	defer globals.RUnlock()
	return globals.testing
}
