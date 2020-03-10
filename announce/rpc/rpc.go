// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"bytes"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/helper"
	"github.com/bitmark-inc/bitmarkd/announce/parameter"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

const (
	addressLimit = 100
	maxNodeCount = 1000
)

type RPC interface {
	Set(fingerprint.Fingerprint, []byte) error
	Add([]byte, []byte, uint64) bool
	Expire()
	IsSet() bool
	Fetch(start uint64, count int) ([]Entry, uint64, error)
	Self() []byte
}

// TODO: rename RPCEntry => Entry
// Entry type of returned data
type Entry struct {
	Fingerprint fingerprint.Fingerprint `json:"fingerprint"`
	Connections []*util.Connection      `json:"connections"`
}

type node struct {
	address   util.PackedConnection   // packed addresses
	fin       fingerprint.Fingerprint // SHA3-256(certificate)
	timestamp time.Time               // creation time
	local     bool                    // true => never expires
}

type rpc struct {
	sync.RWMutex
	fin   fingerprint.Fingerprint
	set   bool
	nodes []*node
	index map[fingerprint.Fingerprint]int
	self  []byte
}

// Set- set this node's rpc announcement data
func (r *rpc) Set(fin fingerprint.Fingerprint, rpcs []byte) error {
	r.Lock()
	defer r.Unlock()

	if r.set {
		return fault.AlreadyInitialised
	}

	r.fin = fin
	r.self = rpcs
	r.set = true

	// add this nodes data to database
	r.add(fin, rpcs, uint64(time.Now().Unix()), true)

	return nil
}

// AddRPC - add an remote RPC listener
// returns:
//   true  if this was a new/updated entry
//   false if the update was within the limits (to prevent continuous relaying)
func (r *rpc) Add(f []byte, rpcs []byte, timestamp uint64) bool {
	var fp fingerprint.Fingerprint
	// discard invalid records
	if len(fp) != len(f) || len(rpcs) > addressLimit {
		return false
	}
	copy(fp[:], f)

	r.Lock()
	rc := r.add(fp, rpcs, timestamp, false)
	r.Unlock()
	return rc
}

// internal add an remote RPC listener, hold lock before calling
func (r *rpc) add(fin fingerprint.Fingerprint, rpcs []byte, timestamp uint64, local bool) bool {
	i, ok := r.index[fin]

	// if new item
	if !ok {
		ts := helper.ResetFutureTimestampToNow(timestamp)
		if helper.IsExpiredAfterDuration(ts, parameter.ExpiryInterval) {
			return false
		}

		// ***** FIX THIS: add more validation here
		e := &node{
			address:   rpcs,
			fin:       fin,
			timestamp: ts,
			local:     local,
		}

		n := len(r.nodes)
		r.nodes = append(r.nodes, e)
		r.index[fin] = n
		return true
	}

	e := r.nodes[i]
	// update old item
	if !bytes.Equal(e.address, rpcs) {
		e.address = rpcs
	}

	// check for too frequent update
	rc := time.Since(e.timestamp) > parameter.RebroadcastInterval

	e.timestamp = time.Now()

	return rc
}

// called in background to expire old RPC entries
// hold lock before calling
func (r *rpc) Expire() {

	n := len(r.nodes)

expiration:
	for i := n - 1; i >= 0; i-- {

		e := r.nodes[i]
		if nil == e || e.local {
			continue expiration
		}

		if time.Since(e.timestamp) > parameter.ExpiryInterval {

			delete(r.index, e.fin)
			n--
			if i != n {
				e := r.nodes[n]
				r.nodes[i] = e
				r.index[e.fin] = i
			}
			r.nodes[n] = nil
		}
	}
	r.nodes = r.nodes[:n] // shrink the list
}

func (r *rpc) IsSet() bool {
	return r.set
}

// Fetch- fetch some records
func (r *rpc) Fetch(start uint64, count int) ([]Entry, uint64, error) {
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

	loop:
		for {
			c, n := a.Unpack()
			if 0 == n {
				break loop
			}
			conn = append(conn, c)
			a = a[n:]
		}
		records[i].Fingerprint = r.nodes[start].fin
		records[i].Connections = conn

		start++
	}

	return records, start, nil
}

func (r *rpc) Self() []byte {
	return r.self
}

func New() RPC {
	return &rpc{
		index: make(map[fingerprint.Fingerprint]int, maxNodeCount),
		nodes: make([]*node, 0, maxNodeCount),
	}
}
