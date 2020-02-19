// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	"github.com/bitmark-inc/bitmarkd/announce/rpc"
)

// SetRPC - set this node's rpc announcement data
func SetRPC(fingerprint fingerprint.Type, rpcs []byte) error {
	return globalData.rpcs.Set(fingerprint, rpcs)
}

// FetchRPCs - fetch some records
func FetchRPCs(start uint64, count int) ([]rpc.Entry, uint64, error) {
	return globalData.rpcs.Fetch(start, count)
}
