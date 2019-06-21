// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package constants

import (
	"time"
)

// the time for a pending record to expire
const (
	ReservoirTimeout = 45 * time.Minute
)

// the time for looking back at old payments when starting up
const (
	OldPaymentTime = 24 * time.Hour
)

// the maximum time before unverified asset is expired
const (
	AssetTimeout = 3 * ReservoirTimeout / 2
)

// the time between rebroadcasts of unconfirmed transactions
const (
	RebroadcastInterval = 3 * ReservoirTimeout / 4
)
