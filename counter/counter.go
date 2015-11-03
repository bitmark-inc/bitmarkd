// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package counter

import (
	"sync/atomic"
)

// type to denote a counter that can be synchronously increments or decremented
// just a 64 bit unsigned integer
type Counter uint64

// add 1 to a counter, returns new value
func (ic *Counter) Increment() uint64 {
	return atomic.AddUint64((*uint64)(ic), 1)
}

// subtract 1 from a counter, returns new value
func (ic *Counter) Decrement() uint64 {
	return atomic.AddUint64((*uint64)(ic), ^uint64(0))
}

// returns current value
func (ic *Counter) Uint64() uint64 {
	return atomic.AddUint64((*uint64)(ic), 0)
}

// check if zero
func (ic *Counter) IsZero() bool {
	return 0 == atomic.AddUint64((*uint64)(ic), 0)
}
