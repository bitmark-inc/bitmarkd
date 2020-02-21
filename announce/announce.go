// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	"github.com/bitmark-inc/bitmarkd/announce/rpc"
)

type Announce interface {
	// Set - set this node's rpc announcement data
	Set(fingerprint.Type, []byte) error

	// Fetch- fetch some records
	Fetch(uint64, int) ([]rpc.Entry, uint64, error)
}

func Get() Announce {
	return globalData.rpcs
}
