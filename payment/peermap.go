package payment

import (
	"sync"

	"github.com/btcsuite/btcd/peer"
)

// PeerMap is a map that maintains the pair between addresses and
// bitcoin peers in a thread-safe way.
type PeerMap struct {
	sync.RWMutex
	peers map[string]*peer.Peer
}

func NewPeerMap() *PeerMap {
	return &PeerMap{
		peers: map[string]*peer.Peer{},
	}
}

// Add will add a new peer into the map.
func (m *PeerMap) Add(addr string, p *peer.Peer) {
	m.Lock()
	defer m.Unlock()
	m.peers[addr] = p
}

// Get will return a specific peer from the map.
func (m *PeerMap) Get(addr string) (p *peer.Peer) {
	m.RLock()
	defer m.RUnlock()
	return m.peers[addr]
}

// Exist validate whether an address is in the map.
func (m *PeerMap) Exist(addr string) bool {
	m.RLock()
	defer m.RUnlock()
	_, ok := m.peers[addr]
	return ok
}

// Len returns the len of current map.
func (m *PeerMap) Len() int {
	m.RLock()
	defer m.RUnlock()
	return len(m.peers)
}

// First will return the first peer get from the map iteration.
// Due to the nature of map in golang, the item is random without fixed order.
func (m *PeerMap) First() (p *peer.Peer) {
	m.RLock()
	defer m.RUnlock()
	for _, v := range m.peers {
		return v
	}

	return nil
}

// Range is a wrapper function of the map iteration.
func (m *PeerMap) Range(callback func(key string, value *peer.Peer)) {
	m.RLock()
	defer m.RUnlock()
	for k, v := range m.peers {
		callback(k, v)
	}
}

// Delete will remove a peer by its address.
func (m *PeerMap) Delete(addr string) {
	m.Lock()
	defer m.Unlock()
	delete(m.peers, addr)
}
