// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package discovery

import (
	"encoding/hex"
	"encoding/json"
	"os"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// PeerItem is the basic structure for backup and restore peers
type PeerItem struct {
	PublicKey []byte
	Listeners []byte
	Timestamp uint64
}

// MarshalText is the json marshal function for PeerItem
func (item PeerItem) MarshalText() ([]byte, error) {
	b := []byte{}
	b = append(b, util.ToVarint64(uint64(len(item.PublicKey)))...)
	b = append(b, item.PublicKey...)
	b = append(b, util.ToVarint64(uint64(len(item.Listeners)))...)
	b = append(b, item.Listeners...)
	b = append(b, util.ToVarint64(uint64(item.Timestamp))...)

	output := make([]byte, hex.EncodedLen(len(b)))
	hex.Encode(output, b)
	return output, nil
}

// UnmarshalText is the json unmarshal function for PeerItem
func (item *PeerItem) UnmarshalText(data []byte) error {
	b := make([]byte, hex.DecodedLen(len(data)))
	_, err := hex.Decode(b, data)
	if err != nil {
		return err
	}
	n := 0

	publicKeyLength, publicKeyOffset := util.ClippedVarint64(b[n:], 1, 8192)
	if 0 == publicKeyOffset || 32 != publicKeyLength {
		return fault.NotPublicKey
	}
	publicKey := make([]byte, publicKeyLength)
	n += publicKeyOffset
	copy(publicKey, b[n:n+publicKeyLength])
	n += publicKeyLength

	listenerLength, listenerOffset := util.ClippedVarint64(b[n:], 1, 8192)

	ll := listenerLength / 19
	if 0 == listenerOffset || ll < 1 || ll > 2 {
		return fault.InvalidIpAddress
	}
	listener := make([]byte, listenerLength)
	n += listenerOffset
	copy(listener, b[n:n+listenerLength])
	n += listenerLength

	timestamp, timestampLength := util.FromVarint64(b[n:])
	if 0 == timestampLength {
		return fault.InvalidTimestamp
	}

	item.PublicKey = publicKey
	item.Listeners = listener
	item.Timestamp = timestamp
	return nil
}

// NewPeerItem is to create a PeerItem from peerEntry
func NewPeerItem(peer *peerEntry) *PeerItem {
	if peer == nil {
		return nil
	}
	return &PeerItem{
		PublicKey: peer.publicKey,
		Listeners: peer.listeners,
		Timestamp: uint64(peer.timestamp.Unix()),
	}
}

// PeerList is a list of PeerItem
type PeerList []PeerItem

// backupPeers will backup all peers into a peer file
func backupPeers(peerFile string) error {
	if globalData.peerTree.Count() <= 2 {
		globalData.log.Info("no need to backup. peer nodes are less than two")
		return nil
	}

	var peers PeerList
	lastNode := globalData.peerTree.Last()
	node := globalData.peerTree.First()

	for node != lastNode {
		peer, ok := node.Value().(*peerEntry)
		if ok && len(peer.listeners) > 0 {
			p := NewPeerItem(peer)
			peers = append(peers, *p)
		}
		node = node.Next()
	}
	// backup the last node
	peer, ok := lastNode.Value().(*peerEntry)
	if ok && len(peer.listeners) > 0 {
		p := NewPeerItem(peer)
		peers = append(peers, *p)
	}

	f, err := os.OpenFile(peerFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(peers)
}

// restorePeers will backup peers from a peer file
func restorePeers(peerFile string) error {
	var peers PeerList

	f, err := os.OpenFile(peerFile, os.O_RDONLY, 0600)
	if err != nil {
		// peer file not exist shouldn't return error, for example when starting
		// bitmarkd first time, peer file doesn't exist.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	d := json.NewDecoder(f)
	err = d.Decode(&peers)
	if err != nil {
		return err
	}

	for _, peer := range peers {
		addPeer(peer.PublicKey, peer.Listeners, peer.Timestamp)
	}
	return nil
}
