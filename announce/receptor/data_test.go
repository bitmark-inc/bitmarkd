// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/id"
	"github.com/bitmark-inc/bitmarkd/avl"
	"github.com/gogo/protobuf/proto"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	peerlib "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

const (
	backupFile = "peers"
)

func TestString(t *testing.T) {
	str1 := "/ip4/1.2.3.4/tcp/1234"
	ma1, _ := ma.NewMultiaddr(str1)
	str2 := "/ip6/::1/tcp/5678"
	ma2, _ := ma.NewMultiaddr(str2)
	r := receptor.Data{
		ID:        peerlib.ID("this is a test"),
		Listeners: []ma.Multiaddr{ma1, ma2},
		Timestamp: time.Now(),
	}

	actual := r.String()

	assert.Equal(t, 2, len(actual), "wrong count")
	assert.Equal(t, str1, actual[0], "wrong first addr")
	assert.Equal(t, str2, actual[1], "wrong second addr")
}

func removeBackupFile() {
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		_ = os.Remove(backupFile)
	}
}

func TestBackup(t *testing.T) {
	removeBackupFile()
	defer removeBackupFile()

	tree := avl.New()
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	addr3, _ := ma.NewMultiaddr("/ip6/11:12:13:14::/tcp/11223")
	now := time.Now()

	p1 := &receptor.Data{
		ID:        "p1",
		Listeners: []ma.Multiaddr{addr1},
		Timestamp: now,
	}

	p2 := &receptor.Data{
		ID:        "p2",
		Listeners: []ma.Multiaddr{addr2},
		Timestamp: now,
	}

	p3 := &receptor.Data{
		ID:        "p3",
		Listeners: []ma.Multiaddr{addr3},
		Timestamp: now,
	}

	tree.Insert(id.ID("p1"), p1)
	tree.Insert(id.ID("p2"), p2)
	tree.Insert(id.ID("p3"), p3)

	err := receptor.Backup(backupFile, tree)
	assert.Nil(t, err, "wrong store")

	data, err := ioutil.ReadFile(backupFile)
	assert.Nil(t, err, "peer file read error")

	var peers receptor.List
	err = proto.Unmarshal(data, &peers)
	assert.Nil(t, err, "wrong unmarshal pb")
	assert.Equal(t, 3, len(peers.Receptors), "wrong peer count")
	assert.Equal(t, "p1", string(peers.Receptors[0].ID), "wrong first peer")
	assert.Equal(t, "p2", string(peers.Receptors[1].ID), "wrong second peer")
	assert.Equal(t, "p3", string(peers.Receptors[2].ID), "wrong third peer")
}

func TestBackupWhenCountLessOrEqualThanTwo(t *testing.T) {
	removeBackupFile()
	defer removeBackupFile()

	tree := avl.New()
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	now := time.Now()
	p1 := &receptor.Data{
		ID:        "p1",
		Listeners: []ma.Multiaddr{addr1},
		Timestamp: now,
	}

	p2 := &receptor.Data{
		ID:        "p2",
		Listeners: []ma.Multiaddr{addr2},
		Timestamp: now,
	}

	tree.Insert(id.ID("p1"), p1)
	tree.Insert(id.ID("p2"), p2)

	err := receptor.Backup(backupFile, tree)
	assert.Nil(t, err, "wrong store")
	_, err = os.Stat(backupFile)
	assert.NotNil(t, err, "peer file should not be stored")
}

func TestRestore(t *testing.T) {
	removeBackupFile()
	defer removeBackupFile()

	tree := avl.New()
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	addr3, _ := ma.NewMultiaddr("/ip6/11:12:13:14::/tcp/11223")
	addr4, _ := ma.NewMultiaddr("/ip4/9.8.7.6/tcp/9876")
	now := time.Now()

	p1 := &receptor.Data{
		ID:        "p1",
		Listeners: []ma.Multiaddr{addr1},
		Timestamp: now,
	}

	p2 := &receptor.Data{
		ID:        "p2",
		Listeners: []ma.Multiaddr{addr2},
		Timestamp: now,
	}

	p3 := &receptor.Data{
		ID:        "p3",
		Listeners: []ma.Multiaddr{addr3},
		Timestamp: now,
	}

	p4 := &receptor.Data{
		ID:        "p4",
		Listeners: []ma.Multiaddr{addr4},
		Timestamp: now,
	}

	tree.Insert(id.ID("p1"), p1)
	tree.Insert(id.ID("p2"), p2)
	tree.Insert(id.ID("p3"), p3)
	tree.Insert(id.ID("p4"), p4)

	_ = receptor.Backup(backupFile, tree)

	peers, err := receptor.Restore(backupFile)
	assert.Nil(t, err, "wrong restore")
	assert.Equal(t, 4, len(peers.Receptors), "wrong peer count")
	assert.Equal(t, "p1", string(peers.Receptors[0].ID), "wrong first peer")
	assert.Equal(t, "p2", string(peers.Receptors[1].ID), "wrong second peer")
	assert.Equal(t, "p3", string(peers.Receptors[2].ID), "wrong third peer")
	assert.Equal(t, "p4", string(peers.Receptors[3].ID), "wrong forth peer")
}

func TestRestoreWhenFileNotExist(t *testing.T) {
	_, err := receptor.Restore("not_exist_file")
	assert.Nil(t, err, "wrong file not exist error")
}
