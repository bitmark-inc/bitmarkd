// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package helper_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/helper"
	"github.com/stretchr/testify/assert"
)

func TestResetFutureTimeToNow(t *testing.T) {
	now := time.Now()
	actual := helper.ResetFutureTimeToNow(uint64(now.Add(time.Minute).Unix()))
	assert.Equal(t, true, actual.After(now), "reset time")
	assert.Equal(t, true, actual.Add(-1*time.Millisecond).Before(now), "reset time")
}

func TestResetFutureTimeToNowWhenPast(t *testing.T) {
	prev := time.Now().Add(-1 * time.Minute)
	actual := helper.ResetFutureTimeToNow(uint64(prev.Unix()))
	assert.Equal(t, time.Unix(int64(prev.Unix()), 0), actual, "wrong previous time")
}

func TestIsExpiredAfterDuration(t *testing.T) {
	now := time.Now()
	actual := helper.IsExpiredAfterDuration(now.Add(-59*time.Second), 60*time.Second)
	assert.False(t, actual, "wrong expire")

	actual = helper.IsExpiredAfterDuration(now.Add(-60*time.Second), 59*time.Second)
	assert.True(t, actual, "wrong expire")
}
