// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

// a state type for the thread
type connectorState int

// state of the connector process
const (
	// register to nodes and make outgoing connections
	cStateConnecting connectorState = iota

	// locate node(s) with highest block number
	cStateHighestBlock connectorState = iota

	// read block hashes to check for possible fork
	cStateForkDetect connectorState = iota

	// fetch blocks from current or fork point
	cStateFetchBlocks connectorState = iota

	// rebuild database from fork point (config setting to force total rebuild)
	cStateRebuild connectorState = iota

	// signal resync complete and sample nodes to see if out of sync occurs
	cStateSampling connectorState = iota
)

func (state connectorState) String() string {
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
