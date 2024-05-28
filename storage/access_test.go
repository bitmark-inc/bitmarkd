// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitmark-inc/bitmarkd/storage/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	dbName     = "data-access"
	defaultKey = "key"
)

var (
	db           *leveldb.DB
	trx          *leveldb.Batch
	defaultValue = []byte{'a'}
)

func initialiseVars() {
	trx = new(leveldb.Batch)
	if db == nil {
		db, _ = leveldb.OpenFile(dbName, nil)
	}
}

func newMockCache(t *testing.T) (*mocks.MockCache, *gomock.Controller) {
	ctl := gomock.NewController(t)
	return mocks.NewMockCache(ctl), ctl
}

func setupDummyMockCache(t *testing.T) *mocks.MockCache {
	mockCache, ctl := newMockCache(t)
	defer ctl.Finish()

	mockCache.EXPECT().Get(gomock.Any()).Return([]byte{}, true).AnyTimes()
	mockCache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockCache.EXPECT().Clear().AnyTimes()

	return mockCache
}

func setupTestDataAccess(mockCache *mocks.MockCache) Access {
	return newDA(db, trx, mockCache)
}

func removeDir(dirName string) {
	dirPath, _ := filepath.Abs(dirName)
	_ = os.RemoveAll(dirPath)
}

func teardownTestDataAccess() {
	_ = db.Close()
	removeDir(dbName)
}

func TestBeginShouldErrorWhenAlreadyInTransaction(t *testing.T) {
	mc := setupDummyMockCache(t)
	da := setupTestDataAccess(mc)

	err := da.Begin()
	assert.Equal(t, nil, err, "first time Begin should with not error")

	err = da.Begin()
	assert.NotEqual(t, nil, err, "second time Begin should return error")
}

func TestCommitDidNotUnlockInUse(t *testing.T) {
	mc := setupDummyMockCache(t)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	_ = da.Commit()

	err := da.Begin()
	assert.NotEqual(t, nil, err, "did not reset internal inUse ")
}

func TestCommitResetTransaction(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mc.EXPECT().Clear().AnyTimes()
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	_ = da.Commit()
	da.Abort()

	actual := da.DumpTx()
	assert.Equal(t, 0, len(actual), "Commit did not reset transaction")
}

func TestCommitWriteToDB(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Get(gomock.Any()).Return(defaultValue, false).AnyTimes()
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mc.EXPECT().Clear().AnyTimes()
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	_ = da.Commit()

	actual, _ := da.Get([]byte(defaultKey))
	assert.Equal(t, defaultValue, actual, "commit not write to db")
}

func TestPutActionCached(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Get(gomock.Any()).Return([]byte{}, true).AnyTimes()
	mc.EXPECT().Set(dbPut, defaultKey, defaultValue).Times(1)
	mc.EXPECT().Clear().AnyTimes()
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
}

func TestDeleteActionCached(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Get(gomock.Any()).Return([]byte{}, true).AnyTimes()
	mc.EXPECT().Set(dbPut, "a", []byte{'b'}).Times(1)
	mc.EXPECT().Set(dbDelete, "a", []byte{}).Times(1)
	mc.EXPECT().Clear().AnyTimes()
	da := setupTestDataAccess(mc)

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte{'a'},
		[]byte{'b'},
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	da.Delete(fixture.key)
}

func TestCommitClearsCache(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Get(gomock.Any()).Return([]byte{}, true).AnyTimes()
	mc.EXPECT().Set(dbPut, defaultKey, defaultValue).Times(1)
	mc.EXPECT().Clear().Times(1)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	_ = da.Commit()
	da.Abort()
}

func TestGetActionReadsFromCache(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Get(gomock.Any()).Return(defaultValue, true).Times(1)
	mc.EXPECT().Set(dbPut, defaultKey, defaultValue).Times(1)
	mc.EXPECT().Clear().Times(0)
	da := setupTestDataAccess(mc)

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte(defaultKey),
		defaultValue,
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	actual, _ := da.Get(fixture.key)

	assert.Equal(t, fixture.value, actual, "wrong cached value")
}

func TestGetActionReadDBIfNotInCache(t *testing.T) {
	key := "random"
	value := []byte{'a', 'b', 'c'}

	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Get(gomock.Any()).Return(value, false).Times(1)
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	mc.EXPECT().Clear().Times(1)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(key), value)
	da.Commit()
	da.Abort()
	actual, _ := da.Get([]byte(key))

	assert.Equal(t, value, actual, "db value not set")
}

func TestInUse(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	da := setupTestDataAccess(mc)

	inUse := da.InUse()
	assert.Equal(t, false, inUse, "inUse default not true")

	_ = da.Begin()
	inUse = da.InUse()
	assert.Equal(t, true, inUse, "inUse not set")
}

func TestAbortResetInUse(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	mc.EXPECT().Clear().Times(1)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	da.Abort()

	inUse := da.InUse()
	assert.Equal(t, false, inUse, "inUse is not set")
}

func TestAbortResetBatch(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	mc.EXPECT().Clear().Times(1)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	da.Abort()

	dump := da.DumpTx()
	assert.Equal(t, []byte{}, dump, "batch not reset")
}

func TestAbortResetCache(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	mc.EXPECT().Clear().Times(1)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	da.Abort()
}

func TestHasCached(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Set(dbPut, defaultKey, defaultValue).Times(1)
	mc.EXPECT().Get(defaultKey).Return(defaultValue, true).Times(1)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	has, err := da.Has([]byte(defaultKey))
	assert.Equal(t, true, has, "cannot cached cached key")
	assert.Equal(t, nil, err, "has with error")
}

func TestHasNotCached(t *testing.T) {
	mc, ctl := newMockCache(t)
	defer ctl.Finish()

	mc.EXPECT().Get(gomock.Any()).Return(defaultValue, false).Times(1)
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	mc.EXPECT().Clear().Times(1)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	da.Put([]byte(defaultKey), defaultValue)
	da.Commit()
	da.Abort()
	has, _ := da.Has([]byte(defaultKey))
	assert.Equal(t, true, has, "didn't check db")
}
