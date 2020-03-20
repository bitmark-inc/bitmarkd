// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"encoding/binary"
	"time"

	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
)

// SendRegistration - send a peer registration request to a client channel
func SendRegistration(client zmqutil.Client, fn string) error {
	chain := mode.ChainName()

	// get a big endian timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	return client.Send(fn, chain, []byte(globalData.receptors.ID()), globalData.receptors.SelfListener(), timestamp)
}

// AddPeer - add a peer announcement to the in-memory tree
// returns:
//   true  if this was a new/updated entry
//   false if the update was within the limits (to prevent continuous relaying)
func AddPeer(publicKey []byte, listeners []byte, timestamp uint64) bool {
	return globalData.receptors.Add(publicKey, listeners, timestamp)
}

// GetRandom - fetch the data for a random node in the ring not matching a given public key
func GetRandom(publicKey []byte) ([]byte, []byte, time.Time, error) {
	return globalData.receptors.Random(publicKey)
}

// SetPeer - called by the peering initialisation to set up this
// node's announcement data
func SetSelf(publicKey []byte, listeners []byte) error {
	return globalData.receptors.SetSelf(publicKey, listeners)
}

// GetNext - fetch the data for the next node in the ring for a given public key
func GetNext(publicKey []byte) ([]byte, []byte, time.Time, error) {
	return globalData.receptors.Next(publicKey)
}
