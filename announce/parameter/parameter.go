// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package parameter

import "time"

const (
	//InitialiseInterval  = 2  * time.Minute // startup delay before first send
	InitialiseInterval = 1 * time.Minute // startup delay before first send

	//RebroadcastInterval = 7  * time.Minute // to prevent too frequent rebroadcasts
	RebroadcastInterval = 30 * time.Second // to prevent too frequent rebroadcasts

	//PollingInterval     = 11 * time.Minute // regular polling time
	PollingInterval = 3 * time.Minute

	ExpiryInterval  = 5 * PollingInterval // if no responses received within this time, delete the entry
	MinTreeExpected = 5 + 1               // voting.minimumClients + 1(self)

	ReFetchingInterval = 1 * time.Hour // re-fetching nodes domain

	Protocol = "p2p" // use by lip2p
)
