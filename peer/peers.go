// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer

import (
	"github.com/bitmark-inc/bitmarkd/announce"
	"github.com/bitmark-inc/bitmarkd/gnomon"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// type to hold Peer
type Peer struct {
	log *logger.L
}

// ------------------------------------------------------------

type NeighbourArguments struct {
	Start *gnomon.Cursor
	Count int
}

type NeighbourReply struct {
	Peers     []announce.RecentData
	NextStart *gnomon.Cursor
}

func (t *Peer) List(arguments *NeighbourArguments, reply *NeighbourReply) error {
	if arguments.Count <= 0 {
		arguments.Count = 10
	}
	peers, nextStart, err := announce.RecentPeers(arguments.Start, arguments.Count, announce.TypePeer)
	if nil == err {
		reply.Peers = make([]announce.RecentData, len(peers))
		for i, d := range peers {
			reply.Peers[i] = d.(announce.RecentData)
		}
		reply.NextStart = nextStart
	}
	return err
}

func (t *Peer) RPCs(arguments *NeighbourArguments, reply *NeighbourReply) error {
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

type PeerArguments struct {
	Address     string
	Fingerprint util.FingerprintBytes
}

type PeerReply struct {
	Added           bool
	NeedCertificate bool
}

func (t *Peer) Put(arguments *PeerArguments, reply *PeerReply) error {

	panic("not yet")
	return nil

	// reply.Added = false
	// reply.NeedCertificate = false

	// address, err := CanonicalIPandPort(arguments.Address)
	// if nil != err {
	// 	return err
	// }

	// // already have it?
	// _, err = announce.GetPeer(address, listenType)
	// if nil == err {
	// 	return nil
	// }

	// _, found := announce.GetCertificate(&arguments.Fingerprint)
	// if !found {
	// 	reply.NeedCertificate = true
	// 	return nil
	// }

	// peerData := announce.PeerData{
	// 	Fingerprint: &arguments.Fingerprint,
	// }
	// justAdded, err := announce.AddPeer(address, listenType, &peerData)
	// if nil == err {
	// 	reply.Added = justAdded
	// }
	// return err
}
