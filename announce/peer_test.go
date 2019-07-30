// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/stretchr/testify/assert"
)

const (
	announceExpiry = 55 * time.Minute
)

func TestIsPeerExpiredFromTimeWhenExpired(t *testing.T) {
	former := uint64(time.Now().Add(-2 * announceExpiry).Unix())
	expired := announce.IsPeerExpiredFromTime(former)
	assert.Equal(t, true, expired, "expired")
}

func TestIsPeerExpiredFromTimeWhenNotExpired(t *testing.T) {
	former := uint64(time.Now().Add(-1 * announceExpiry / 2).Unix())
	expired := announce.IsPeerExpiredFromTime(former)
	assert.Equal(t, false, expired, "expired")
}
