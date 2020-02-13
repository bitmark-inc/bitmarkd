// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/bitmark-inc/bitmarkd/util"

	"github.com/bitmark-inc/bitmarkd/announce/id"

	"github.com/bitmark-inc/logger"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/stretchr/testify/assert"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	peer "github.com/libp2p/go-libp2p-core/peer"
)

const (
	testingDirName = "testing"
	logCategory    = "announce"
)

func setupTestLogger() {
	removeFiles()
	_ = os.Mkdir(testingDirName, 0700)

	logging := logger.Configuration{
		Directory: testingDirName,
		File:      "testing.log",
		Size:      1048576,
		Count:     10,
		Console:   false,
		Levels: map[string]string{
			logger.DefaultTag: "critical",
		},
	}

	// start logging
	_ = logger.Initialise(logging)
}

func teardownTestLogger() {
	removeFiles()
}

func removeFiles() {
	_ = os.RemoveAll(testingDirName)
}

func TestAdd(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	now := time.Now()
	myID := peer.ID("test")

	added := r.Add(myID, []ma.Multiaddr{addr1, addr2}, uint64(now.Unix()))
	assert.True(t, added, "not add")

	pid, addr, ts, err := r.Next(myID)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID, pid, "wrong myID")
	assert.Equal(t, 2, len(addr), "wrong addr count")
	assert.Equal(t, time.Unix(now.Unix(), 0), ts, "wrong time")
}

func TestAddWhenExpired(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	expired := time.Now().Add(-20 * time.Minute)
	myID := peer.ID("test")

	added := r.Add(myID, []ma.Multiaddr{addr1, addr2}, uint64(expired.Unix()))
	assert.False(t, added, "not add")
}

func TestAddWhenAddSameItemTooFast(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	now := time.Now()
	myID := peer.ID("test")

	added := r.Add(myID, []ma.Multiaddr{addr1, addr2}, uint64(now.Unix()))
	assert.True(t, added, "not add")

	added = r.Add(myID, []ma.Multiaddr{addr1, addr2}, uint64(now.Unix()))
	assert.False(t, added, "not add")
}

func TestChanged(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	assert.False(t, r.Changed(), "not changed")

	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")

	added := r.Add(peer.ID("test"), []ma.Multiaddr{addr}, uint64(time.Now().Unix()))
	assert.True(t, added, "not add")
	assert.True(t, r.Changed(), "not changed")
}

func TestNext(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	now := time.Now()
	id1 := peer.ID("test1")
	_ = r.Add(id1, []ma.Multiaddr{addr1}, uint64(now.Unix()))

	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	id2 := peer.ID("test2")
	_ = r.Add(id2, []ma.Multiaddr{addr2}, uint64(now.Unix()))

	pid, addr, ts, err := r.Next(id1)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, id2, pid, "wrong id")
	assert.Equal(t, 1, len(addr), "wrong addr count")
	assert.Equal(t, time.Unix(now.Unix(), 0), ts, "wrong time")
}

func TestRandom(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	now := time.Now()
	id1 := peer.ID("test1")
	_ = r.Add(id1, []ma.Multiaddr{addr1}, uint64(now.Unix()))

	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	id2 := peer.ID("test2")
	_ = r.Add(id2, []ma.Multiaddr{addr2}, uint64(now.Unix()))

	pid, addr, ts, err := r.Random(id1)
	assert.Nil(t, err, "wrong random")
	assert.Equal(t, id2, pid, "wrong id")
	assert.Equal(t, 1, len(addr), "wrong addr count")
	assert.Equal(t, time.Unix(now.Unix(), 0), ts, "wrong time")
}

func TestRandomWhenNotAbleToFindDifferent(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	now := time.Now()
	id1 := peer.ID("test1")
	_ = r.Add(id1, []ma.Multiaddr{addr1}, uint64(now.Unix()))

	pid, addr, _, err := r.Random(id1)
	assert.Equal(t, fault.InvalidPublicKey, err, "wrong random")
	assert.Equal(t, peer.ID(""), pid, "wrong id")
	assert.Equal(t, 0, len(addr), "wrong addr count")
}

func TestSetSelf(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	assert.False(t, r.IsSet(), "wrong set")

	myID := peer.ID("test")
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")

	err := r.SetSelf(myID, []ma.Multiaddr{addr})
	assert.Nil(t, err, "wrong SetSelf")
	assert.True(t, r.IsSet(), "wrong set")

	pid, addrs, _, err := r.Next(myID)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID, pid, "wrong pid")
	assert.Equal(t, 1, len(addrs), "wrong addrs")
}

func TestTree(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	tree := r.Tree()
	assert.Equal(t, 0, tree.Count(), "wrong tree node count")

	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	now := time.Now()
	myID := peer.ID("test")

	_ = r.Add(myID, []ma.Multiaddr{addr}, uint64(now.Unix()))
	tree = r.Tree()
	assert.Equal(t, 1, tree.Count(), "wrong tree node count")
}

func TestID(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	pid := peer.ID("test")
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	_ = r.SetSelf(pid, []ma.Multiaddr{addr})

	assert.Equal(t, pid, r.ID(), "wrong ID")
}

func TestSelf(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	pid := peer.ID("test")
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	_ = r.SetSelf(pid, []ma.Multiaddr{addr})

	s := r.Self()
	assert.Equal(t, 0, s.Key().Compare(id.ID(pid)), "wrong self")
}

func TestChange(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	assert.False(t, r.Changed(), "wrong change")

	r.Change(true)
	assert.True(t, r.Changed(), "wrong change")
}

func TestSelfAddress(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	pid := peer.ID("test")
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	_ = r.SetSelf(pid, []ma.Multiaddr{addr})

	addrs := r.SelfAddress()
	assert.Equal(t, 1, len(addrs), "wrong address count")
	assert.True(t, addrs[0].Equal(addr), "wrong address")
}

func TestAddrToString(t *testing.T) {
	raw := "/ip4/1.2.3.4/tcp/1234"
	addr, _ := ma.NewMultiaddr(raw)
	binAddr := util.GetBytesFromMultiaddr([]ma.Multiaddr{addr})
	pbAddr, _ := proto.Marshal(&receptor.Addrs{Address: binAddr})
	str := receptor.AddrToString(pbAddr)
	assert.Equal(t, raw, strings.Trim(str, "\n"), "wrong to string")
}

func TestBinaryID(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	myID := peer.ID("test")
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	_ = r.SetSelf(myID, []ma.Multiaddr{addr})

	binID, _ := myID.MarshalBinary()
	assert.Equal(t, binID, r.BinaryID(), "wrong binary ID")
}

func TestShortID(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	myID := peer.ID("test")
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	_ = r.SetSelf(myID, []ma.Multiaddr{addr})

	assert.Equal(t, myID.ShortString(), r.ShortID(), "wrong short ID")
}

func TestUpdateTime(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	myID := peer.ID("test")
	addr, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	_ = r.SetSelf(myID, []ma.Multiaddr{addr})

	future := time.Now().Add(5 * time.Minute)
	r.UpdateTime(myID, future)
	pid, addrs, timestamp, err := r.Next(myID)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID, pid, "wrong id")
	assert.Equal(t, 1, len(addrs), "wrong addrs count")
	assert.Equal(t, future, timestamp, "wrong updated time")
}

func TestExpire(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	r := receptor.New(logger.New(logCategory))
	addr1, _ := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/1234")
	addr2, _ := ma.NewMultiaddr("/ip6/5:6:7:8::/tcp/5678")
	now := time.Now()
	expired := now.Add(-899 * time.Second) // expire 15 minutes = 900 seconds
	myID1 := peer.ID("test1")
	myID2 := peer.ID("test2")

	_ = r.Add(myID1, []ma.Multiaddr{addr1}, uint64(now.Unix()))
	_ = r.Add(myID2, []ma.Multiaddr{addr2}, uint64(expired.Unix()))

	tree := r.Tree()
	assert.Equal(t, 2, tree.Count(), "wrong tree count")

	time.Sleep(time.Second)
	r.Expire()
	assert.Equal(t, 1, tree.Count(), "wrong expire")
	pid, addrs, _, err := r.Next(myID1)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID1, pid, "wrong id")
	assert.Equal(t, 1, len(addrs), "wrong addrs count")
	assert.True(t, addrs[0].Equal(addr1), "wrong address")
}

// TODO: test BalanceTree
