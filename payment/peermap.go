// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"sync"

	"github.com/btcsuite/btcd/peer"
)

// PeerMap is a map that maintains the pair between addresses and
// bitcoin peers in a thread-safe way.
type PeerMap struct {
	mu    sync.RWMutex
	peers map[string]*peer.Peer
}

func NewPeerMap() *PeerMap {
	return &PeerMap{
		peers: map[string]*peer.Peer{},
	}
}

// Add will add a new peer into the map.
func (m *PeerMap) Add(addr string, p *peer.Peer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.peers[addr] = p
}

// Get will return a specific peer from the map.
func (m *PeerMap) Get(addr string) (p *peer.Peer) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.peers[addr]
}

// Exist validate whether an address is in the map.
func (m *PeerMap) Exist(addr string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.peers[addr]
	return ok
}

// Len returns the len of current map.
func (m *PeerMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.peers)
}

// First will return the first peer get from the map iteration.
// Due to the nature of map in golang, the item is random without fixed order.
func (m *PeerMap) First() (p *peer.Peer) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, v := range m.peers {
		return v
	}

	return nil
}

// Range is a wrapper function of the map iteration.
func (m *PeerMap) Range(callback func(key string, value *peer.Peer)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for k, v := range m.peers {
		callback(k, v)
	}
}

// Delete will remove a peer by its address.
func (m *PeerMap) Delete(addr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.peers, addr)
}
