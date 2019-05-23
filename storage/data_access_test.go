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
	dbName        = "data-access"
	getDefaultKey = "key"
)

var (
	db              *leveldb.DB
	trx             *leveldb.Batch
	getDefaultValue = []byte{'a'}
)

func initialiseVars() {
	trx = new(leveldb.Batch)
	if nil == db {
		db, _ = leveldb.OpenFile(dbName, nil)
	}
}

func newMockCache(t *testing.T) *mocks.MockCache {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	return mocks.NewMockCache(ctl)
}

func setupDummyMockCache(t *testing.T) *mocks.MockCache {
	mockCache := newMockCache(t)
	mockCache.EXPECT().Get(gomock.Any()).Return([]byte{}, true).AnyTimes()
	mockCache.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockCache.EXPECT().Clear().AnyTimes()

	return mockCache
}

func setupTestDataAccess(mockCache *mocks.MockCache) DataAccess {
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

func TestMain(m *testing.M) {
	initialiseVars()
	result := m.Run()
	teardownTestDataAccess()
	os.Exit(result)
}

func TestBeginShouldErrorWhenAlreadyInTransaction(t *testing.T) {
	mc := setupDummyMockCache(t)
	da := setupTestDataAccess(mc)

	err := da.Begin()
	assert.Equal(t, nil, err, "first time Begin should with not error")

	err = da.Begin()
	assert.NotEqual(t, nil, err, "second time Begin should return error")
}

func TestCommitUnlockInUse(t *testing.T) {
	mc := setupDummyMockCache(t)
	da := setupTestDataAccess(mc)

	_ = da.Begin()
	_ = da.Commit()

	err := da.Begin()
	assert.Equal(t, nil, err, "did not reset internal inUse ")
}

func TestCommitResetTransaction(t *testing.T) {
	mc := newMockCache(t)
	mc.EXPECT().Get(gomock.Any()).Return([]byte{'b'}, true).Times(1)
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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
	_ = da.Commit()

	actual := da.DumpTx()
	assert.Equal(t, 0, len(actual), "Commit did not reset transaction")
}

func TestCommitWriteToDB(t *testing.T) {
	mc := newMockCache(t)
	mc.EXPECT().Get(gomock.Any()).Return([]byte{'b'}, true).AnyTimes()
	mc.EXPECT().Set(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
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
	_ = da.Commit()

	actual, _ := da.Get(fixture.key)
	assert.Equal(t, fixture.value, actual, "commit not write to db")
}

func TestPutActionCached(t *testing.T) {
	mc := newMockCache(t)
	mc.EXPECT().Get(gomock.Any()).Return([]byte{}, true).AnyTimes()
	mc.EXPECT().Set(dbPut, "a", []byte{'b'}).Times(1)
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
}

func TestDeleteActionCached(t *testing.T) {
	mc := newMockCache(t)
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
	mc := newMockCache(t)
	mc.EXPECT().Get(gomock.Any()).Return([]byte{}, true).AnyTimes()
	mc.EXPECT().Set(dbPut, "a", []byte{'b'}).Times(1)
	mc.EXPECT().Clear().Times(1)
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
	_ = da.Commit()
}

func TestGetActionReadsFromCache(t *testing.T) {
	mc := newMockCache(t)
	mc.EXPECT().Get(gomock.Any()).Return(getDefaultValue, true).Times(1)
	mc.EXPECT().Set(dbPut, getDefaultKey, getDefaultValue).Times(1)
	mc.EXPECT().Clear().Times(0)
	da := setupTestDataAccess(mc)

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte(getDefaultKey),
		getDefaultValue,
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	actual, _ := da.Get(fixture.key)

	assert.Equal(t, fixture.value, actual, "wrong cached value")
}

func TestGetActionReadDBIfNotInCache(t *testing.T) {
	key := "random"
	value := []byte{'a', 'b', 'c'}

	mc := newMockCache(t)
	mc.EXPECT().Get(gomock.Any()).Return(value, true).Times(1)
	mc.EXPECT().Set(dbPut, key, value).Times(1)
	mc.EXPECT().Clear().Times(1)
	da := setupTestDataAccess(mc)

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte(key),
		value,
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	da.Commit()
	actual, _ := da.Get(fixture.key)

	assert.Equal(t, fixture.value, actual, "db value not set")
}
