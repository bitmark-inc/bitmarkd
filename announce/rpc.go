// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import ()

type connection struct {
	ip   net.IP
	port int
}

type fingerprint [32]byte

type rpcEntry struct {
	address     []connection
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

// add an RPC listener
func AddRPC(address string, fingerprint [32]byte) {
	globalData.Lock()
	e := &rpcEntry{
		address:     []byte(address),
		fingerprint: fingerprint,
	}
	globalData.rpcs = append(globalData.rpcs, e)
	globalData.Unlock()
}
