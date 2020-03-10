// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/avl"
)

type Entity struct {
	PublicKey []byte
	Listeners []byte
	Timestamp time.Time // last seen time
}

type StoreEntity struct {
	PublicKey []byte
	Listeners []byte
	Timestamp uint64 // last seen time
}

// MarshalText is the json marshal function for PeerItem
func (e StoreEntity) MarshalText() ([]byte, error) {
	var b []byte
	b = append(b, util.ToVarint64(uint64(len(e.PublicKey)))...)
	b = append(b, e.PublicKey...)
	b = append(b, util.ToVarint64(uint64(len(e.Listeners)))...)
	b = append(b, e.Listeners...)
	b = append(b, util.ToVarint64(e.Timestamp)...)

	output := make([]byte, hex.EncodedLen(len(b)))
	hex.Encode(output, b)
	return output, nil
}

// UnmarshalText is the json unmarshal function for PeerItem
func (e *StoreEntity) UnmarshalText(data []byte) error {
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

	e.PublicKey = publicKey
	e.Listeners = listener
	e.Timestamp = timestamp
	return nil
}

// Backup store all peers into a backup file
func Backup(peerFile string, tree *avl.Tree) error {
	if tree.Count() <= 2 {
		return nil
	}

	var list []StoreEntity
	lastNode := tree.Last()
	node := tree.First()

	for node != lastNode {
		n, ok := node.Value().(*Entity)
		if ok && len(n.Listeners) > 0 {
			e := StoreEntity{
				PublicKey: n.PublicKey,
				Listeners: n.Listeners,
				Timestamp: uint64(n.Timestamp.Unix()),
			}
			list = append(list, e)
		}
		node = node.Next()
	}

	// backup the last node
	n, ok := lastNode.Value().(*Entity)
	if ok && len(n.Listeners) > 0 {
		e := StoreEntity{
			PublicKey: n.PublicKey,
			Listeners: n.Listeners,
			Timestamp: uint64(n.Timestamp.Unix()),
		}
		list = append(list, e)
	}

	f, err := os.OpenFile(peerFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(list)
}

// Restore peers from a backup file
func Restore(peerFile string, r Receptor) error {
	var list []StoreEntity

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
	err = d.Decode(&list)
	if err != nil {
		return err
	}

	for _, peer := range list {
		_ = r.Add(peer.PublicKey, peer.Listeners, peer.Timestamp)
	}
	return nil
}
