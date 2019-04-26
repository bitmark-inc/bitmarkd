// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"time"

	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/fault"
)

// limiting for a single request
func rateLimit(limiter *rate.Limiter) error {
	r := limiter.Reserve()
	if !r.OK() {
		return fault.ErrRateLimiting
	}
	time.Sleep(r.Delay())
	return nil
}

// limiting for a multiple request
func rateLimitN(limiter *rate.Limiter, count int, maximumCount int) error {

	// invalid count gets limited as a single request
	if count <= 0 || count > maximumCount {

		r := limiter.Reserve()
		if !r.OK() {
			return fault.ErrRateLimiting
		}
		time.Sleep(r.Delay())

		return fault.ErrInvalidCount
	}

	r := limiter.ReserveN(time.Now(), count)
	if !r.OK() {
		return fault.ErrRateLimiting
	}
	time.Sleep(r.Delay())

	return nil
}
