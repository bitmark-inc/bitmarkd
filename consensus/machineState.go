// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package consensus

// a state type for the thread
type state int

// state of the connector process
const (
	// register to nodes and make outgoing connections
	cStateConnecting state = iota

	// locate node(s) with highest block number
	cStateHighestBlock state = iota

	// read block hashes to check for possible fork
	cStateForkDetect state = iota

	// fetch blocks from current or fork point
	cStateFetchBlocks state = iota

	// rebuild database from fork point (config setting to force total rebuild)
	cStateRebuild state = iota

	// signal resync complete and sample nodes to see if out of sync occurs
	cStateSampling state = iota
)

func (state state) String() string {
	switch state {
	case cStateConnecting:
		return "Connecting"
	case cStateHighestBlock:
		return "HighestBlock"
	case cStateForkDetect:
		return "ForkDetect"
	case cStateFetchBlocks:
		return "FetchBlocks"
	case cStateRebuild:
		return "Rebuild"
	case cStateSampling:
		return "Sampling"
	default:
		return "*Unknown*"
	}
}
