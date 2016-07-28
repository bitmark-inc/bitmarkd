// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"github.com/bitmark-inc/bitmarkd/fault"
)

// type peerEntry struct {
// 	publicKey  []byte
// 	broadcasts []byte
// 	listen     []byte
// }

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

	return nil
}
