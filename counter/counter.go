// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package counter

import (
	"sync/atomic"
)

// Counter - type to denote a counter that can be synchronously increments or decremented
// just a 64 bit unsigned integer
type Counter uint64

// Increment - add 1 to a counter, returns new value
func (ic *Counter) Increment() uint64 {
	return atomic.AddUint64((*uint64)(ic), 1)
}

// Decrement - subtract 1 from a counter, returns new value
func (ic *Counter) Decrement() uint64 {
	return atomic.AddUint64((*uint64)(ic), ^uint64(0))
}

// Uint64 - returns current value
func (ic *Counter) Uint64() uint64 {
	return atomic.AddUint64((*uint64)(ic), 0)
}

// IsZero - check if zero
func (ic *Counter) IsZero() bool {
	return atomic.AddUint64((*uint64)(ic), 0) == 0
}
