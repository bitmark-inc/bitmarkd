// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/helper"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

type RPC interface {
	Add([]byte, []byte, uint64) bool
	Expire()
	Fetch(uint64, int) ([]Entry, uint64, error)
	IsSet() bool
	Self() []byte
	Set(fingerprint.Type, []byte) error
}

// Entry type of returned data
type Entry struct {
	Fingerprint fingerprint.Type   `json:"fingerprint"`
	Connections []*util.Connection `json:"connections"`
}

type node struct {
	address     util.PackedConnection // packed addresses
	fingerprint fingerprint.Type      // SHA3-256(certificate)
	timestamp   time.Time             // creation time
	local       bool                  // true => never expires
}

type rpc struct {
	sync.RWMutex
	self        []byte
	set         bool
	fingerprint fingerprint.Type

	// database of all RPCs
	// TODO: rename
	rpcIndex map[fingerprint.Type]int // index to find rpc node
	nodes    []*node                  // array of RPCs
}

// New - return RPC interface
func New() RPC {
	return &rpc{
		rpcIndex: make(map[fingerprint.Type]int, 1000),
		nodes:    make([]*node, 0, 1000),
	}
}

func (r rpc) IsSet() bool {
	return r.set
}

func (r rpc) Self() []byte {
	return r.self
}

// Set - set this node's rpc announcement data
func (r *rpc) Set(fingerprint fingerprint.Type, rpcs []byte) error {
	r.Lock()
	defer r.Unlock()

	if r.set {
		return fault.AlreadyInitialised
	}
	r.fingerprint = fingerprint
	r.self = rpcs
	r.set = true

	// add this nodes data to database
	r.add(fingerprint, rpcs, uint64(time.Now().Unix()), true)

	return nil
}

// Add - add an remote RPC listener
// returns:
//   true  if this was a new/updated node
//   false if the update was within the limits (to prevent continuous relaying)
func (r *rpc) Add(clientFingerprint []byte, rpcs []byte, timestamp uint64) bool {
	var fp fingerprint.Type
	// discard invalid records
	// TODO: 100 should be constant
	if len(fp) != len(clientFingerprint) || len(rpcs) > 100 {
		return false
	}
	copy(fp[:], clientFingerprint)

	r.Lock()
	rc := r.add(fp, rpcs, timestamp, false)
	r.Unlock()
	return rc
}

// internal add an remote RPC listener, hold lock before calling
func (r *rpc) add(fingerprint fingerprint.Type, rpcs []byte, timestamp uint64, local bool) bool {

	i, ok := r.rpcIndex[fingerprint]

	// if new item
	if !ok {
		ts := helper.ResetFutureTimeToNow(timestamp)
		// TODO: setup this by other way, previous in announcer package
		if helper.IsExpiredAfterDuration(ts, 15*time.Minute) {
			return false
		}

		// ***** FIX THIS: add more validation here
		e := &node{
			address:     rpcs,
			fingerprint: fingerprint,
			timestamp:   ts,
			local:       local,
		}
		n := len(r.nodes)
		r.nodes = append(r.nodes, e)
		r.rpcIndex[fingerprint] = n
		return true
	}

	e := r.nodes[i]
	// update old item
	if !bytes.Equal(e.address, rpcs) {
		e.address = rpcs
	}

	// check for too frequent update
	// TODO: is this necessary? previous was 7 minute
	rc := time.Since(e.timestamp) > 30*time.Second

	e.timestamp = time.Now()

	return rc
}

// Expire - expire outdated nodes
// called in background to expire old RPC nodes
// hold lock before calling
func (r *rpc) Expire() {
	n := len(r.nodes)
loop:
	for i := n - 1; i >= 0; i -= 1 {

		e := r.nodes[i]
		if nil == e || e.local {
			continue loop
		}

		// TODO: fix this
		if time.Since(e.timestamp) > 15*time.Minute {
			delete(r.rpcIndex, e.fingerprint)
			n -= 1
			if i != n {
				e := r.nodes[n]
				r.nodes[i] = e
				r.rpcIndex[e.fingerprint] = i
			}
			r.nodes[n] = nil
		}
	}
	r.nodes = r.nodes[:n] // shrink the nodes
}

// Fetch - fetch some records
func (r rpc) Fetch(start uint64, count int) ([]Entry, uint64, error) {
	if count <= 0 {
		return nil, 0, fault.InvalidCount
	}

	r.Lock()
	defer r.Unlock()

	n := uint64(len(r.nodes))
	if start >= n {
		return nil, 0, nil
	}

	remainder := n - start
	c := uint64(count)

	if c >= remainder {
		c = remainder
	}

	records := make([]Entry, c)
	for i := uint64(0); i < c; i += 1 {

		a := r.nodes[start].address

		conn := make([]*util.Connection, 0, 4)

	innerLoop:
		for {
			c, n := a.Unpack()
			if 0 == n {
				break innerLoop
			}
			conn = append(conn, c)
			a = a[n:]
		}
		records[i].Fingerprint = r.nodes[start].fingerprint
		records[i].Connections = conn

		start++
	}

	return records, start, nil
}
