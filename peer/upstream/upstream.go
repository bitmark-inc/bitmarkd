// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package upstream

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
)

// ActiveInThePast - active upstream in past time
func (u *upstreamData) ActiveInThePast(d time.Duration) bool {
	now := time.Now()
	limit := now.Add(-d)
	active := limit.Before(u.lastResponseTime)
	difference := now.Sub(u.lastResponseTime).Seconds()

	u.log.Debugf("active: %t, last response time %s, difference %f seconds",
		active,
		u.lastResponseTime.Format("2006-01-02, 15:04:05 -0700"),
		difference,
	)
	return active
}

// Destroy - shutdown a connection and terminate its background processes
func (u *upstreamData) Destroy() {
	if u != nil {
		close(u.shutdown)
	}
}

// ResetServer - clear Server side info of Zmq client for reusing the
// upstream
func (u *upstreamData) ResetServer() {
	u.client.Close()
	u.connected = false
	u.remoteHeight = 0
}

// IsConnectedTo - check the current destination
//
// does not mean actually connected, as could be in a timeout and
// reconnect state
func (u *upstreamData) IsConnectedTo(serverPublicKey []byte) bool {
	return u.client.IsConnectedTo(serverPublicKey)
}

// IsConnected - check if registered and have a valid connection
func (u *upstreamData) IsConnected() bool {
	return u.connected
}

// ConnectedTo - if registered return the connection data
func (u *upstreamData) ConnectedTo() *zmqutil.Connected {
	return u.client.ConnectedTo()
}

// Connect - connect (or reconnect) to a specific server
func (u *upstreamData) Connect(address *util.Connection, serverPublicKey []byte) error {
	u.log.Infof("connecting to address: %s", address)
	u.log.Infof("connecting to server: %x", serverPublicKey)
	return u.client.Connect(address, serverPublicKey, mode.ChainName())
}

// ServerPublicKey - return the internal ZeroMQ client data
func (u *upstreamData) ServerPublicKey() []byte {
	return u.client.ServerPublicKey()
}

// RemoteDigestOfHeight - fetch block digest from a specific block number
func (u *upstreamData) RemoteDigestOfHeight(blockNumber uint64) (blockdigest.Digest, error) {
	remoteAddr, _ := u.RemoteAddr()
	u.log.Debugf("remote address %s, get block digest %d", remoteAddr, blockNumber)

	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("H", parameter)
	if err == nil {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if err != nil {
		return blockdigest.Digest{}, err
	}

	if len(data) != 2 {
		return blockdigest.Digest{}, fault.InvalidPeerResponse
	}

	switch string(data[0]) {
	case "E":
		return blockdigest.Digest{}, fault.BlockNotFound
	case "H":
		d := blockdigest.Digest{}
		if blockdigest.Length == len(data[1]) {
			err := blockdigest.DigestFromBytes(&d, data[1])
			return d, err
		}
	default:
	}
	return blockdigest.Digest{}, fault.InvalidPeerResponse
}

// GetBlockData - fetch block data from a specific block number
// Note: returned data is always nil for error conditions
func (u *upstreamData) GetBlockData(blockNumber uint64) ([]byte, error) {

	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("B", parameter)
	if err == nil {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if err != nil {
		return nil, err
	}

	if len(data) != 2 {
		return nil, fault.InvalidPeerResponse
	}

	switch string(data[0]) {
	case "E":
		return nil, fault.BlockNotFound
	case "B":
		return data[1], nil
	default:
	}
	return nil, fault.InvalidPeerResponse
}

// must have lock held before calling
func (u *upstreamData) RemoteHeight() (uint64, error) {
	u.log.Infof("RemoteHeight: client: %s", u.client)

	err := u.client.Send("N")
	if err != nil {
		u.log.Errorf("RemoteHeight: %s send error: %s", u.client, err)
		return 0, err
	}

	data, err := u.client.Receive(0)
	if err != nil {
		u.log.Errorf("RemoteHeight: %s receive error: %s", u.client, err)
		return 0, err
	}
	if len(data) != 2 {
		return 0, fmt.Errorf("RemoteHeight: received: %d  expected: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return 0, fmt.Errorf("RemoteHeight: error response: %q", data[1])
	case "N":
		if len(data[1]) != 8 {
			return 0, fmt.Errorf("RemoteHeight: invalid response: %q", data[1])
		}
		height := binary.BigEndian.Uint64(data[1])
		u.log.Infof("height: %d", height)
		return height, nil
	default:
		return 0, fmt.Errorf("RemoteHeight: unexpected response: %q", data[0])
	}
}

// CachedRemoteDigestOfLocalHeight - cached remote digest of local block height
func (u *upstreamData) CachedRemoteDigestOfLocalHeight() blockdigest.Digest {
	return u.remoteDigestOfLocalHeight
}

// RemoteAddr - remote client address
func (u *upstreamData) RemoteAddr() (string, error) {
	var err error

	if u.client == nil {
		err = fault.ClientSocketNotCreated
	} else if !u.client.IsConnected() {
		err = fault.ClientSocketNotConnected
	}
	if err != nil {
		u.log.Warnf("remote address not available error: %s", err)
		return "", err
	}

	return u.client.String(), nil
}

// Name - upstream name
func (u *upstreamData) Name() string {
	return u.name
}

// CachedRemoteHeight - cached remote client height
func (u *upstreamData) CachedRemoteHeight() uint64 {
	return u.remoteHeight
}

// LocalHeight - local block height
func (u *upstreamData) LocalHeight() uint64 {
	return u.localHeight
}
