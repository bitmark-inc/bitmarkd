// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package announce

import (
	"bytes"
	"encoding/gob"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/bitmark-inc/bitmarkd/pool"
	"github.com/bitmark-inc/bitmarkd/util"
)

// type of listener
const (
	TypeRPC  = iota
	TypePeer = iota
)

// a type to store data about a peer
type PeerData struct {
	Fingerprint *util.FingerprintBytes
}

// add a peer announcement to the corresponding LRU
func AddPeer(address string, listenType int, data *PeerData) (bool, error) {

	newlyAdded := false
	buffer := bytes.Buffer{}
	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(data)
	if nil != err {
		return newlyAdded, err
	}

	switch listenType {
	case TypePeer:
		newlyAdded, err = announce.peerPool.Add([]byte(address), buffer.Bytes())
	case TypeRPC:
		newlyAdded, err = announce.rpcPool.Add([]byte(address), buffer.Bytes())
	default:
		err = fault.ErrInvalidType
	}
	return newlyAdded, err
}

// look up a p2p neighbour in the LRU
func GetPeer(address string, listenType int) (*PeerData, error) {

	var data []byte
	var err error

	switch listenType {
	case TypePeer:
		data, err = announce.peerPool.Get([]byte(address))
	case TypeRPC:
		data, err = announce.rpcPool.Get([]byte(address))
	default:
		err = fault.ErrInvalidType
	}
	if nil != err {
		return nil, err
	}

	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	result := PeerData{}
	decoder.Decode(&result)
	return &result, nil
}

// type to hold recent data
type RecentData struct {
	Address string
	Data    *PeerData
}

// get a the most recently anounced addresses of a given type
func RecentPeers(start *gnomon.Cursor, count int, listenType int) ([]interface{}, *gnomon.Cursor, error) {
	if nil == start {
		start = &gnomon.Cursor{}
	}

	if count <= 0 {
		return nil, nil, fault.ErrInvalidCount
	}

	var thePool *pool.IndexedPool
	switch listenType {
	case TypePeer:
		thePool = announce.peerPool
	case TypeRPC:
		thePool = announce.rpcPool
	default:
		return nil, nil, fault.ErrInvalidType
	}

	recent, nextStart, err := thePool.Recent(start, count, func(key []byte, value []byte) interface{} {
		buffer := bytes.NewBuffer(value)
		decoder := gob.NewDecoder(buffer)
		result := PeerData{}
		decoder.Decode(&result)

		return RecentData{
			Address: string(key),
			Data:    &result,
		}
	})
	if nil != err {
		return nil, nil, err
	}

	return recent, nextStart, nil
}
