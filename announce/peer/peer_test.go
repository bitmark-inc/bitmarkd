// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package peer_test

import (
	fmt "fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/bitmark-inc/bitmarkd/announce/id"

	"github.com/bitmark-inc/bitmarkd/avl"

	"github.com/bitmark-inc/bitmarkd/announce/peer"

	p2pPeer "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce/receiver"

	ma "github.com/multiformats/go-multiaddr"
)

const (
	peerFile = "peers"
)

func TestNewPeerItem(t *testing.T) {
	now := time.Now()
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5555:6666:7777:8888::/tcp/5678")
	str := "test"

	r := receiver.Receiver{
		ID:        p2pPeer.ID(str),
		Listeners: []ma.Multiaddr{addr1, addr2},
		Timestamp: now,
	}
	actual := peer.NewPeerItem(&r)
	assert.Equal(t, []byte(str), actual.PeerID, "wrong str")
	assert.Equal(t, 2, len([][]byte(actual.Listeners.Address)), "wrong listener length")
	assert.Equal(t, addr1.Bytes(), [][]byte(actual.Listeners.Address)[0], "wrong first ip")
	assert.Equal(t, addr2.Bytes(), [][]byte(actual.Listeners.Address)[1], "wrong second ip")
	assert.Equal(t, uint64(now.Unix()), actual.Timestamp, "wrong time")
}

func TestNewPeerItemWhenPeerIsNil(t *testing.T) {
	actual := peer.NewPeerItem(nil)
	assert.Nil(t, actual, "wrong nil peer")
}

func removePeerFile() {
	if _, err := os.Stat(peerFile); !os.IsNotExist(err) {
		_ = os.Remove(peerFile)
	}
}

func TestBackup(t *testing.T) {
	removePeerFile()
	defer removePeerFile()

	tree := avl.New()
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	addr3, _ := ma.NewMultiaddr("/ip6/11:12:13:14::/tcp/11223")
	now := time.Now()
	p1 := &receiver.Receiver{
		ID:        "p1",
		Listeners: []ma.Multiaddr{addr1},
		Timestamp: now,
	}

	p2 := &receiver.Receiver{
		ID:        "p2",
		Listeners: []ma.Multiaddr{addr2},
		Timestamp: now,
	}

	p3 := &receiver.Receiver{
		ID:        "p3",
		Listeners: []ma.Multiaddr{addr3},
		Timestamp: now,
	}

	tree.Insert(id.ID("p1"), p1)
	tree.Insert(id.ID("p2"), p2)
	tree.Insert(id.ID("p3"), p3)

	err := peer.Backup(peerFile, tree)
	assert.Nil(t, err, "wrong store")

	data, err := ioutil.ReadFile(peerFile)
	assert.Nil(t, err, "peer file read error")

	var peers peer.PeerList
	err = proto.Unmarshal(data, &peers)
	assert.Nil(t, err, "wrong unmarshal pb")
	assert.Equal(t, 3, len(peers.Peers), "wrong peer count")
	assert.Equal(t, "p1", string(peers.Peers[0].PeerID), "wrong first peer")
	assert.Equal(t, "p2", string(peers.Peers[1].PeerID), "wrong second peer")
	assert.Equal(t, "p3", string(peers.Peers[2].PeerID), "wrong third peer")
}

func TestBackupWhenCountLessOrEqualThanTwo(t *testing.T) {
	removePeerFile()
	defer removePeerFile()

	tree := avl.New()
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	now := time.Now()
	p1 := &receiver.Receiver{
		ID:        "p1",
		Listeners: []ma.Multiaddr{addr1},
		Timestamp: now,
	}

	p2 := &receiver.Receiver{
		ID:        "p2",
		Listeners: []ma.Multiaddr{addr2},
		Timestamp: now,
	}

	tree.Insert(id.ID("p1"), p1)
	tree.Insert(id.ID("p2"), p2)

	fmt.Println("count: ", tree.Count())

	err := peer.Backup(peerFile, tree)
	assert.Nil(t, err, "wrong store")
	_, err = os.Stat(peerFile)
	assert.NotNil(t, err, "peer file should not be stored")
}

func TestRestore(t *testing.T) {
	removePeerFile()
	defer removePeerFile()

	tree := avl.New()
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	addr3, _ := ma.NewMultiaddr("/ip6/11:12:13:14::/tcp/11223")
	addr4, _ := ma.NewMultiaddr("/ip4/9.8.7.6/tcp/9876")
	now := time.Now()

	p1 := &receiver.Receiver{
		ID:        "p1",
		Listeners: []ma.Multiaddr{addr1},
		Timestamp: now,
	}

	p2 := &receiver.Receiver{
		ID:        "p2",
		Listeners: []ma.Multiaddr{addr2},
		Timestamp: now,
	}

	p3 := &receiver.Receiver{
		ID:        "p3",
		Listeners: []ma.Multiaddr{addr3},
		Timestamp: now,
	}

	p4 := &receiver.Receiver{
		ID:        "p4",
		Listeners: []ma.Multiaddr{addr4},
		Timestamp: now,
	}

	tree.Insert(id.ID("p1"), p1)
	tree.Insert(id.ID("p2"), p2)
	tree.Insert(id.ID("p3"), p3)
	tree.Insert(id.ID("p4"), p4)

	_ = peer.Backup(peerFile, tree)

	peers, err := peer.Restore(peerFile)
	assert.Nil(t, err, "wrong restore")
	assert.Equal(t, 4, len(peers.Peers), "wrong peer count")
	assert.Equal(t, "p1", string(peers.Peers[0].PeerID), "wrong first peer")
	assert.Equal(t, "p2", string(peers.Peers[1].PeerID), "wrong second peer")
	assert.Equal(t, "p3", string(peers.Peers[2].PeerID), "wrong third peer")
	assert.Equal(t, "p4", string(peers.Peers[3].PeerID), "wrong forth peer")
}

func TestRestoreWhenFileNotExist(t *testing.T) {
	_, err := peer.Restore("not_exist_file")
	assert.Nil(t, err, "wrong file not exist error")
}
