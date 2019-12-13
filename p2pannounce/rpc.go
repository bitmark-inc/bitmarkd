// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package p2pannounce

import (
	"bytes"
	"time"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// SetRPC - set this node's rpc announcement data
func SetRPC(fingerprint fingerprintType, rpcs []byte) error {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.rpcsSet {
		return fault.AlreadyInitialised
	}
	globalData.fingerprint = fingerprint
	globalData.rpcs = rpcs
	globalData.rpcsSet = true

	// add this nodes data to database
	addRPC(fingerprint, rpcs, 0, true)

	return nil
}

// AddRPC - add an remote RPC listener
// returns:
//   true  if this was a new/updated entry
//   false if the update was within the limits (to prevent continuous relaying)
func AddRPC(fingerprint []byte, rpcs []byte, timestamp uint64) bool {

	var fp fingerprintType
	// discard invalid records
	if len(fp) != len(fingerprint) || len(rpcs) > 100 {
		return false
	}
	copy(fp[:], fingerprint)

	globalData.Lock()
	rc := addRPC(fp, rpcs, timestamp, false)
	globalData.Unlock()
	return rc
}

// internal add an remote RPC listener, hold lock before calling
func addRPC(fingerprint fingerprintType, rpcs []byte, timestamp uint64, local bool) bool {

	i, ok := globalData.rpcIndex[fingerprint]

	// if new item
	if !ok {
		ts := resetFutureTimestampToNow(timestamp)
		if isPeerExpiredFromTime(ts) {
			return false
		}

		// ***** FIX THIS: add more validation here
		e := &rpcEntry{
			address:     rpcs,
			fingerprint: fingerprint,
			timestamp:   ts,
			local:       local,
		}
		n := len(globalData.rpcList)
		globalData.rpcList = append(globalData.rpcList, e)
		globalData.rpcIndex[fingerprint] = n
		return true
	}

	e := globalData.rpcList[i]
	// update old item
	if !bytes.Equal(e.address, rpcs) {
		e.address = rpcs
	}

	// check for too frequent update
	rc := time.Since(e.timestamp) > announceRebroadcast

	e.timestamp = time.Now()

	return rc
}

// called in background to expire old RPC entries
// hold lock before calling
func expireRPC() {

	n := len(globalData.rpcList)
expirations:
	for i := n - 1; i >= 0; i -= 1 {

		e := globalData.rpcList[i]
		if nil == e || e.local {
			continue expirations
		}

		if time.Since(e.timestamp) > announceExpiry {

			delete(globalData.rpcIndex, e.fingerprint)
			n -= 1
			if i != n {
				e := globalData.rpcList[n]
				globalData.rpcList[i] = e
				globalData.rpcIndex[e.fingerprint] = i
			}
			globalData.rpcList[n] = nil
		}
	}
	globalData.rpcList = globalData.rpcList[:n] // shrink the list
}

// RPCEntry type of returned data
type RPCEntry struct {
	Fingerprint fingerprintType    `json:"fingerprint"`
	Connections []*util.Connection `json:"connections"`
}

// FetchRPCs - fetch some records
func FetchRPCs(start uint64, count int) ([]RPCEntry, uint64, error) {
	if count <= 0 {
		return nil, 0, fault.InvalidCount
	}

	globalData.Lock()
	defer globalData.Unlock()

	n := uint64(len(globalData.rpcList))
	if start >= n {
		return nil, 0, nil
	}

	remainder := n - start
	c := uint64(count)

	if c >= remainder {
		c = remainder
	}

	records := make([]RPCEntry, c)
	for i := uint64(0); i < c; i += 1 {

		a := globalData.rpcList[start].address

		conn := make([]*util.Connection, 0, 4)

	inner_loop:
		for {
			c, n := a.Unpack()
			if 0 == n {
				break inner_loop
			}
			conn = append(conn, c)
			a = a[n:]
		}
		records[i].Fingerprint = globalData.rpcList[start].fingerprint
		records[i].Connections = conn

		start += 1
	}

	return records, start, nil
}
