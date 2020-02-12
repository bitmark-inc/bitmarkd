// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package helper

import "time"

// ResetFutureTimeToNow - reset future time to now
func ResetFutureTimeToNow(timestamp uint64) time.Time {
	ts := time.Unix(int64(timestamp), 0)
	now := time.Now()
	if now.Before(ts) {
		return now
	}
	return ts
}

// IsExpiredAfterDuration - is peer expired from time
func IsExpiredAfterDuration(ts time.Time, dur time.Duration) bool {
	return ts.Add(dur).Before(time.Now())
}
