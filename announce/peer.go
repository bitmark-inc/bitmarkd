// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"bytes"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/fault"
)

type pubkey []byte

type peerEntry struct {
	publicKey  []byte
	broadcasts []byte
	listeners  []byte
}

// set this node's peer announcement data
func SetPeer(publicKey []byte, broadcasts []byte, listeners []byte) error {
	globalData.Lock()
	defer globalData.Unlock()

	if globalData.peerSet {
		return fault.ErrAlreadyInitialised
	}
	globalData.publicKey = publicKey
	globalData.broadcasts = broadcasts
	globalData.listeners = listeners
	globalData.peerSet = true

	addPeer(publicKey, broadcasts, listeners)

	return nil
}

// add a peer announcement to the in-memory tree
func AddPeer(publicKey []byte, broadcasts []byte, listeners []byte) {
	globalData.Lock()
	addPeer(publicKey, broadcasts, listeners)
	globalData.Unlock()
}

// internal add a peer announcement, hold lock before calling
func addPeer(publicKey []byte, broadcasts []byte, listeners []byte) {
	peer := &peerEntry{
		publicKey:  publicKey,
		broadcasts: broadcasts,
		listeners:  listeners,
	}
	globalData.peerTree.Insert(pubkey(publicKey), peer)
	globalData.peerTree.Print(false)
}

// public key comparison
func (p pubkey) Compare(q interface{}) int {
	return bytes.Compare(p, q.(pubkey))
}
func (p pubkey) String() string {
	return fmt.Sprintf("%x", []byte(p))
}
