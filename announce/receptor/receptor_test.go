// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package receptor_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce/fixtures"
	"github.com/bitmark-inc/bitmarkd/announce/id"
	"github.com/bitmark-inc/bitmarkd/announce/parameter"
	"github.com/bitmark-inc/bitmarkd/announce/receptor"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

func TestAdd(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	listeners := []byte{1, 2}
	now := time.Now()
	myID := []byte("test")

	added := r.Add(myID, listeners, uint64(now.Unix()))
	assert.True(t, added, "not add")

	pid, addr, ts, err := r.Next(myID)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID, pid, "wrong myID")
	assert.Equal(t, 2, len(addr), "wrong addr count")
	assert.Equal(t, time.Unix(now.Unix(), 0), ts, "wrong time")
}

func TestAddWhenExpired(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	listeners := []byte{1, 2}
	expired := time.Now().Add(-60 * time.Minute)
	myID := []byte("test")

	added := r.Add(myID, listeners, uint64(expired.Unix()))
	assert.False(t, added, "not add")
}

func TestAddWhenAddSameItemTooFast(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	listeners := []byte{1, 2}
	now := time.Now()
	myID := []byte("test")

	added := r.Add(myID, listeners, uint64(now.Unix()))
	assert.True(t, added, "not add")

	added = r.Add(myID, listeners, uint64(now.Unix()))
	assert.False(t, added, "not add")
}

func TestChanged(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	assert.False(t, r.IsChanged(), "not changed")

	listeners := []byte{1, 2}

	added := r.Add([]byte("test"), listeners, uint64(time.Now().Unix()))
	assert.True(t, added, "not add")
	assert.True(t, r.IsChanged(), "not changed")
}

func TestNext(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	listeners1 := []byte{1, 2}
	now := time.Now()
	id1 := []byte("test1")
	_ = r.Add(id1, listeners1, uint64(now.Unix()))

	listeners2 := []byte{1}
	id2 := []byte("test2")
	_ = r.Add(id2, listeners2, uint64(now.Unix()))

	pid, addr, ts, err := r.Next(id1)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, id2, pid, "wrong id")
	assert.Equal(t, 1, len(addr), "wrong addr count")
	assert.Equal(t, time.Unix(now.Unix(), 0), ts, "wrong time")
}

func TestRandom(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	listeners1 := []byte{1, 2}
	now := time.Now()
	id1 := []byte("test1")
	_ = r.Add(id1, listeners1, uint64(now.Unix()))

	listeners2 := []byte{3}
	id2 := []byte("test2")
	_ = r.Add(id2, listeners2, uint64(now.Unix()))

	listeners3 := []byte{4}
	id3 := []byte("test3")
	_ = r.Add(id3, listeners3, uint64(now.Unix()))

	pid, addr, ts, err := r.Random(id1)
	assert.Nil(t, err, "wrong random")
	assert.NotEqual(t, id1, pid, "wrong id")
	assert.Equal(t, 1, len(addr), "wrong addr count")
	assert.Equal(t, time.Unix(now.Unix(), 0), ts, "wrong time")
}

func TestRandomWhenNotAbleToFindDifferent(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	listeners := []byte{1, 2}
	now := time.Now()
	id1 := []byte("test1")
	_ = r.Add(id1, listeners, uint64(now.Unix()))

	pid, addr, _, err := r.Random(id1)
	assert.Equal(t, fault.InvalidPublicKey, err, "wrong random")
	assert.Equal(t, []byte{}, pid, "wrong id")
	assert.Equal(t, []byte{}, addr, "wrong addr count")
}

func TestSetSelf(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	assert.False(t, r.IsInitialised(), "wrong initialised")

	myID := []byte("test")
	listeners := []byte{1, 2}

	err := r.SetSelf(myID, listeners)
	assert.Nil(t, err, "wrong SetSelf")
	assert.True(t, r.IsInitialised(), "wrong initialised")

	pid, addrs, _, err := r.Next(myID)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID, pid, "wrong pid")
	assert.Equal(t, 2, len(addrs), "wrong addrs")
}

func TestConnectable(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	tree := r.Connectable()
	assert.Equal(t, 0, tree.Count(), "wrong tree node count")

	listeners := []byte{1, 2}
	now := time.Now()
	myID := []byte("test")

	_ = r.Add(myID, listeners, uint64(now.Unix()))
	tree = r.Connectable()
	assert.Equal(t, 1, tree.Count(), "wrong tree node count")
}

func TestID(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	pid := []byte("test")
	listeners := []byte{1, 2}
	_ = r.SetSelf(pid, listeners)

	assert.Equal(t, id.ID(pid), r.ID(), "wrong ID")
}

func TestSelf(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	pid := []byte("test")
	listeners := []byte{1, 2}
	_ = r.SetSelf(pid, listeners)

	s := r.Self()
	assert.Equal(t, 0, s.Key().Compare(id.ID(pid)), "wrong self")
}

func TestChange(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	assert.False(t, r.IsChanged(), "wrong change")

	r.Change(true)
	assert.True(t, r.IsChanged(), "wrong change")
}

func TestSelfAddress(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	pid := []byte("test")
	listeners := []byte{1, 2}
	_ = r.SetSelf(pid, listeners)

	addrs := r.SelfListener()
	assert.Equal(t, 2, len(addrs), "wrong address count")
	assert.Equal(t, listeners[0], addrs[0], "wrong address")
	assert.Equal(t, listeners[1], addrs[1], "wrong address")
}

func TestUpdateTime(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	myID := []byte("test")
	listeners := []byte{1, 2}
	_ = r.SetSelf(myID, listeners)

	future := time.Now().Add(5 * time.Minute)
	r.UpdateTime(myID, future)
	pid, addrs, timestamp, err := r.Next(myID)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID, pid, "wrong id")
	assert.Equal(t, 2, len(addrs), "wrong addrs count")
	assert.Equal(t, future, timestamp, "wrong updated time")
}

func TestExpire(t *testing.T) {
	fixtures.SetupTestLogger()
	defer fixtures.TeardownTestLogger()

	r := receptor.New(logger.New(fixtures.LogCategory))
	listeners1 := []byte{1, 2}
	listeners2 := []byte{3}
	now := time.Now()
	expired := now.Add(-1 * (parameter.ExpiryInterval - time.Second)) // expire 55 minutes = 3300 seconds
	myID1 := []byte("test1")
	myID2 := []byte("test2")

	_ = r.Add(myID1, listeners1, uint64(now.Unix()))
	_ = r.Add(myID2, listeners2, uint64(expired.Unix()))

	tree := r.Connectable()
	assert.Equal(t, 2, tree.Count(), "wrong tree count")

	time.Sleep(time.Second)
	r.Expire()
	assert.Equal(t, 1, tree.Count(), "wrong expire")
	pid, addrs, _, err := r.Next(myID1)
	assert.Nil(t, err, "wrong next")
	assert.Equal(t, myID1, pid, "wrong id")
	assert.Equal(t, 2, len(addrs), "wrong addrs count")
	assert.Equal(t, listeners1[0], addrs[0], "wrong address")
}

// TODO: test ReBalance
