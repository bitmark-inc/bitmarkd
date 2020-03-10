// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc_test

import (
	"testing"
	"time"

	"github.com/bitmark-inc/bitmarkd/announce/parameter"

	"github.com/bitmark-inc/bitmarkd/fault"

	"github.com/stretchr/testify/assert"

	"github.com/bitmark-inc/bitmarkd/announce/fingerprint"
	"github.com/bitmark-inc/bitmarkd/announce/rpc"
)

func TestSet(t *testing.T) {
	r := rpc.New()
	f := fingerprint.Fingerprint{1, 2, 3, 4}
	startIndex := uint64(0)

	err := r.Set(f, []byte{5, 6, 7, 8})
	assert.Nil(t, err, "wrong set")

	entries, start, err := r.Fetch(startIndex, 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, startIndex+1, start, "wrong start")
	assert.Equal(t, 1, len(entries), "wrong entries count")
	assert.Equal(t, f, entries[0].Fingerprint, "wrong fingerprint")
}

func TestSetWhenRepeat(t *testing.T) {
	r := rpc.New()
	f := fingerprint.Fingerprint{1, 2, 3, 4}

	err := r.Set(f, []byte{5, 6, 7, 8})
	assert.Nil(t, err, "wrong set")

	err = r.Set(f, []byte{5, 6, 7, 8})
	assert.Equal(t, fault.AlreadyInitialised, err, "wrong second set")
}

func TestAdd(t *testing.T) {
	r := rpc.New()
	fp := make([]byte, 32)
	fp[0] = 1
	b := []byte{6, 7, 8, 9}
	startIndex := uint64(0)

	added := r.Add(fp, b, uint64(time.Now().Unix()))
	assert.True(t, added, "wrong add")

	rpcs, start, err := r.Fetch(startIndex, 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, 1, len(rpcs), "wrong list")
	assert.Equal(t, startIndex+1, start, "")
	assert.Equal(t, fingerprint.Fingerprint{1}, rpcs[0].Fingerprint, "wrong fingerprint")
}

func TestAddWhenExpired(t *testing.T) {
	r := rpc.New()
	fp := make([]byte, 32)
	fp[0] = 1
	b := []byte{6, 7, 8, 9}

	added := r.Add(fp, b, uint64(0))
	assert.False(t, added, "wrong add")
}

func TestAddWhenInvalidFingerprint(t *testing.T) {
	r := rpc.New()
	fp := make([]byte, 30)
	fp[0] = 1
	b := []byte{6, 7, 8, 9}

	added := r.Add(fp, b, uint64(time.Now().Unix()))
	assert.False(t, added, "wrong add")
}

func TestAddWhenInvalidRPC(t *testing.T) {
	r := rpc.New()
	fp := make([]byte, 32)
	fp[0] = 1
	b := make([]byte, 200)
	b[0] = 5

	added := r.Add(fp, b, uint64(time.Now().Unix()))
	assert.False(t, added, "wrong add")
}

func TestFetchWhenStartTooLarge(t *testing.T) {
	r := rpc.New()
	fp := make([]byte, 32)
	fp[0] = 1
	added := r.Add(fp, []byte{6, 7, 8, 9, 10}, uint64(time.Now().Unix()))
	assert.True(t, added, "wrong add")

	nodes, start, err := r.Fetch(uint64(5), 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, 0, len(nodes), "wrong list")
	assert.Equal(t, uint64(0), start, "wrong start")
}

func TestFetchWhenCountLessZero(t *testing.T) {
	r := rpc.New()
	fp := make([]byte, 32)
	fp[0] = 1
	added := r.Add(fp, []byte{6, 7, 8, 9, 10}, uint64(time.Now().Unix()))
	assert.True(t, added, "wrong add")

	_, _, err := r.Fetch(uint64(5), -1)
	assert.Equal(t, fault.InvalidCount, err, "wrong fetch")
}

func TestExpire(t *testing.T) {
	r := rpc.New()
	fp1 := make([]byte, 32)
	fp1[0] = 1
	b := []byte{6, 7, 8, 9}
	now := time.Now()
	startIndex := uint64(0)

	_ = r.Add(fp1, b, uint64(now.Unix()))

	fp2 := make([]byte, 32)
	fp2[0] = 2
	expiredTime := now.Add(-1 * (parameter.ExpiryInterval - time.Second))
	_ = r.Add(fp2, b, uint64(expiredTime.Unix()))

	entries, start, err := r.Fetch(startIndex, 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, 2, len(entries), "wrong entry count")
	assert.Equal(t, startIndex+2, start, "wrong start")

	time.Sleep(time.Second)
	r.Expire()

	entries, start, err = r.Fetch(startIndex, 10)
	assert.Nil(t, err, "wrong fetch")
	assert.Equal(t, 1, len(entries), "wrong entry count")
	assert.Equal(t, startIndex+1, start, "wrong start")
}

func TestIsSet(t *testing.T) {
	r := rpc.New()
	assert.False(t, r.IsSet(), "wrong set")

	err := r.Set(fingerprint.Fingerprint{1, 2, 3, 4}, []byte{5, 6, 7, 8})
	assert.Nil(t, err, "wrong set")

	assert.True(t, r.IsSet(), "wrong set")
}

func TestSelf(t *testing.T) {
	r := rpc.New()
	b := []byte{5, 6, 7, 8}
	_ = r.Set(fingerprint.Fingerprint{1, 2, 3, 4}, b)

	assert.Equal(t, b, r.Self(), "wrong self")
}
