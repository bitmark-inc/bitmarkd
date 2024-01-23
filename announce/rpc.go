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
func SetRPC(fin fingerprint.Fingerprint, listeners []byte) error {
	return globalData.rpcs.Set(fin, listeners)
}

// AddRPC - add an remote RPC listener
// returns:
//
//	true  if this was a new/updated entry
//	false if the update was within the limits (to prevent continuous relaying)
func AddRPC(fin []byte, listeners []byte, timestamp uint64) bool {
	return globalData.rpcs.Add(fin, listeners, timestamp)
}

// FetchRPCs - fetch some records
func FetchRPCs(start uint64, count int) ([]rpc.Entry, uint64, error) {
	return globalData.rpcs.Fetch(start, count)
}
