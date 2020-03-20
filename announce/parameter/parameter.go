// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package parameter

import "time"

const (
	InitialiseInterval  = 2 * time.Minute       // startup delay before first send
	PollingInterval     = 11 * time.Minute      // regular polling time
	BroadcastInterval   = 11 * time.Minute      // regular polling time
	ExpiryInterval      = 5 * BroadcastInterval // if no responses received within this time, delete the entry
	RebroadcastInterval = 7 * time.Minute       // to prevent too frequent rebroadcasts
)
