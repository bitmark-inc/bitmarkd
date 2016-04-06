// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// type to hold RPCs
type RPCs struct {
	log *logger.L
}

// ------------------------------------------------------------

type RpcListArguments struct {
	Start *gnomon.Cursor
	Count int
}

type RpcListReply struct {
	Peers     []announce.RecentData
	NextStart *gnomon.Cursor
}

func (t *RPCs) List(arguments *RpcListArguments, reply *RpcListReply) error {
	if arguments.Count <= 0 {
		arguments.Count = 10
	}
	peers, nextStart, err := announce.RecentPeers(arguments.Start, arguments.Count, announce.TypeRPC)
	if nil == err {
		reply.Peers = make([]announce.RecentData, len(peers))
		for i, d := range peers {
			reply.Peers[i] = d.(announce.RecentData)
		}
		reply.NextStart = nextStart
	}
	return err
}

// ------------------------------------------------------------

type RpcPutArguments struct {
	Address     string
	Fingerprint util.FingerprintBytes
}

type RpcPutReply struct {
	Added           bool
	NeedCertificate bool
}

func (t *RPCs) Put(arguments *RpcPutArguments, reply *RpcPutReply) error {

	reply.Added = false
	reply.NeedCertificate = false

	address, err := util.CanonicalIPandPort(arguments.Address)
	if nil != err {
		return err
	}

	// already have it?
	_, err = announce.GetPeer(address, announce.TypeRPC)
	if nil == err {
		return nil
	}

	if !announce.HasCertificate(&arguments.Fingerprint) {
		reply.NeedCertificate = true
		return nil
	}

	peerData := announce.PeerData{
		Fingerprint: &arguments.Fingerprint,
	}
	justAdded, err := announce.AddPeer(address, announce.TypeRPC, &peerData)
	if nil == err {
		reply.Added = justAdded
	}
	return err
}
