// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"net"
)

type connection struct {
	ip   net.IP
	port uint16
}

type peerEntry struct {
	subscribe []connection
	connect   []connection
}

type peers map[publicKey]peerEntry

type broadcastEntry struct {
	address   []byte //[]connection
	publicKey []byte
}

// add an broadcaster
func AddBroadcast(address string, publicKey []byte) {
	globalData.Lock()
	e := &broadcastEntry{
		address:   []byte(address),
		publicKey: publicKey,
	}
	globalData.broadcasts = append(globalData.broadcasts, e)
	globalData.Unlock()
}

type listenEntry struct {
	address   []byte
	publicKey []byte
}

// add an listener
func AddListen(address string, publicKey []byte) {
	globalData.Lock()
	e := &listenEntry{
		address:   []byte(address),
		publicKey: publicKey,
	}
	globalData.listeners = append(globalData.listeners, e)
	globalData.Unlock()
}
