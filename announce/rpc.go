// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

type fingerprint [32]byte

type rpcEntry struct {
	address     []byte
	fingerprint fingerprint
}

// number of most recent entries to keep
const rpcQueueSize = 500

// index to find entry
var rpcIndex map[fingerprint]*rpcEntry

// fixed size circular queue
var rpcQueue [rpcQueueSize]*rpcEntry

// how to have timestamp order?
// linear search?

// set this node's rpc announcement data
func SetRPC(fingerprint [32]byte, rpcs []byte) error {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.rpcsSet {
		return fault.ErrAlreadyInitialised
	}
	globalData.fingerprint = fingerprint
	globalData.rpcs = rpcs
	globalData.rpcsSet = true

	return nil
}

// add an RPC listener
func AddRPC(fingerprint [32]byte, rpcs []byte) {
	globalData.Lock()
	// e := &rpcEntry{
	// 	//address:     address, // ***** FIX THIS: ?
	// 	fingerprint: fingerprint,
	// }
	// //globalData.rpcs = append(globalData.rpcs, e) // ***** FIX THIS: ?
	globalData.Unlock()
}
