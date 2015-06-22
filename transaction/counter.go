// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"sync/atomic"
)

// type to denote a counter that can be synchronouslt increments or decremented
// just a 64 bit unsigned integer
type ItemCounter uint64


// add 1 to a counter, returns new value
func (ic *ItemCounter) Increment() uint64 {
	return atomic.AddUint64((*uint64)(ic), 1)
}

// subtract 1 from a counter, returns new value
func (ic *ItemCounter) Decrement() uint64 {
	return atomic.AddUint64((*uint64)(ic), ^uint64(0))
}

// returns current value
func (ic *ItemCounter) Uint64() uint64 {
	return atomic.AddUint64((*uint64)(ic), 0)
}
