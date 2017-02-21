// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package constants

import (
	"time"
)

// the maximum time before either a payment track or proof is received
// if the timeout is reached then the transactions are dropped
const (
	PaymentTimeout = 2 * time.Hour
)

// the maximum time before unverified asset is expired
const (
	AssetTimeout = 2 * time.Hour
)

// the time for a record to expire
const (
	ReservoirTimeout = 2 * time.Hour
)
