// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package upstream

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/logger"
)

type UpstreamIntf interface {
	ActiveInPastSeconds(time.Duration) bool
	CachedRemoteDigestOfLocalHeight() blockdigest.Digest
	CachedRemoteHeight() uint64
	Client() zmqutil.ClientIntf
	Connect(*util.Connection, []byte) error
	ConnectedTo() *zmqutil.Connected
	Destroy()
	GetBlockData(uint64) ([]byte, error)
	IsConnectedTo([]byte) bool
	IsConnected() bool
	LocalHeight() uint64
	Name() string
	RemoteAddr() (string, error)
	RemoteDigestOfHeight(uint64) (blockdigest.Digest, error)
	RemoteHeight() (uint64, error)
	ResetServer()
	ServerPublicKey() []byte
}

// Upstream - structure to hold an upstream connection
type Upstream struct {
	sync.RWMutex

	UpstreamIntf

	log                       *logger.L
	name                      string
	client                    zmqutil.ClientIntf
	connected                 bool
	remoteHeight              uint64
	localHeight               uint64
	remoteDigestOfLocalHeight blockdigest.Digest
	shutdown                  chan<- struct{}
	lastResponseTime          time.Time
}

// ActiveInPastSeconds - active upstream in past seconds
func (u *Upstream) ActiveInPastSeconds(sec time.Duration) bool {
	now := time.Now()
	limit := now.Add(time.Second * sec * -1)
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
func (u *Upstream) Destroy() {
	if nil != u {
		close(u.shutdown)
	}
}

// ResetServer - clear Server side info of Zmq client for reusing the
// upstream
func (u *Upstream) ResetServer() {
	u.client.Close()
	u.connected = false
	u.remoteHeight = 0
}

// IsConnectedTo - check the current destination
//
// does not mean actually connected, as could be in a timeout and
// reconnect state
func (u *Upstream) IsConnectedTo(serverPublicKey []byte) bool {
	return u.client.IsConnectedTo(serverPublicKey)
}

// IsConnected - check if registered and have a valid connection
func (u *Upstream) IsConnected() bool {
	return u.connected
}

// ConnectedTo - if registered return the connection data
func (u *Upstream) ConnectedTo() *zmqutil.Connected {
	return u.client.ConnectedTo()
}

// Connect - connect (or reconnect) to a specific server
func (u *Upstream) Connect(address *util.Connection, serverPublicKey []byte) error {
	u.log.Infof("connecting to address: %s", address)
	u.log.Infof("connecting to server: %x", serverPublicKey)
	return u.client.Connect(address, serverPublicKey, mode.ChainName())
}

// ServerPublicKey - return the internal ZeroMQ client data
func (u *Upstream) ServerPublicKey() []byte {
	return u.client.ServerPublicKey()
}

// RemoteDigestOfHeight - fetch block digest from a specific block number
func (u *Upstream) RemoteDigestOfHeight(blockNumber uint64) (blockdigest.Digest, error) {
	remoteAddr, _ := u.RemoteAddr()
	u.log.Debugf("remote address %s, get block digest %d", remoteAddr, blockNumber)

	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("H", parameter)
	if nil == err {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if nil != err {
		return blockdigest.Digest{}, err
	}

	if 2 != len(data) {
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
func (u *Upstream) GetBlockData(blockNumber uint64) ([]byte, error) {
	parameter := make([]byte, 8)
	binary.BigEndian.PutUint64(parameter, blockNumber)

	// critical section - lock out the runner process
	u.Lock()
	var data [][]byte
	err := u.client.Send("B", parameter)
	if nil == err {
		data, err = u.client.Receive(0)
	}
	u.Unlock()

	if nil != err {
		return nil, err
	}

	if 2 != len(data) {
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
func (u *Upstream) RemoteHeight() (uint64, error) {
	u.log.Infof("getHeight: client: %s", u.client)

	err := u.client.Send("N")
	if nil != err {
		u.log.Errorf("getHeight: %s send error: %s", u.client, err)
		return 0, err
	}

	data, err := u.client.Receive(0)
	if nil != err {
		u.log.Errorf("push: %s receive error: %s", u.client, err)
		return 0, err
	}
	if 2 != len(data) {
		return 0, fmt.Errorf("getHeight received: %d  expected: 2", len(data))
	}

	switch string(data[0]) {
	case "E":
		return 0, fmt.Errorf("rpc error response: %q", data[1])
	case "N":
		if 8 != len(data[1]) {
			return 0, fmt.Errorf("highestBlock: rpc invalid response: %q", data[1])
		}
		height := binary.BigEndian.Uint64(data[1])
		u.log.Infof("height: %d", height)
		return height, nil
	default:
		return 0, fmt.Errorf("rpc unexpected response: %q", data[0])
	}
}

// CachedRemoteDigestOfLocalHeight - cached remote digest of local block height
func (u *Upstream) CachedRemoteDigestOfLocalHeight() blockdigest.Digest {
	return u.remoteDigestOfLocalHeight
}

// RemoteAddr - remote client address
func (u *Upstream) RemoteAddr() (string, error) {
	var err error

	if nil == u.client {
		err = fmt.Errorf("client not created")
	} else if !u.client.IsConnected() {
		err = fmt.Errorf("client socket not connected")
	}

	if nil != err {
		u.log.Error(err.Error())
		return "", err
	}

	return u.client.String(), nil
}

// Name - upstream name
func (u *Upstream) Name() string {
	return u.name
}

// CachedRemoteHeight - cached remote client height
func (u *Upstream) CachedRemoteHeight() uint64 {
	return u.remoteHeight
}

// LocalHeight - local block height
func (u *Upstream) LocalHeight() uint64 {
	return u.localHeight
}

// Client - zmq client
func (u *Upstream) Client() zmqutil.ClientIntf {
	return u.client
}
