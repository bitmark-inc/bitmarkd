// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package helper

import "time"

// ResetFutureTimestampToNow - reset future timestamp to now
func ResetFutureTimestampToNow(timestamp uint64) time.Time {
	ts := time.Unix(int64(timestamp), 0)
	now := time.Now()
	if now.Before(ts) {
		return now
	}
	return ts
}

// IsExpiredAfterDuration- is peer expired from time
func IsExpiredAfterDuration(timestamp time.Time, d time.Duration) bool {
	return timestamp.Add(d).Before(time.Now())
}
