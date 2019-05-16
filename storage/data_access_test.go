package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

const (
	dbName        = "data-access"
	getDefaultKey = "key"
)

var (
	setCalled       = false
	getCalled       = false
	clearCalled     = false
	db              *leveldb.DB
	getDefaultValue = []byte{'a'}
)

type fakeCache struct{}

func (f *fakeCache) Get(key string) ([]byte, bool) {
	getCalled = true
	if getDefaultKey == key {
		return getDefaultValue, true
	}
	return []byte{}, false
}
func (f *fakeCache) Set(dbOperation, string, []byte) {
	setCalled = true
}
func (f *fakeCache) Clear() {
	clearCalled = true
}

func initializeLevelDB() {
	if nil == db {
		db, _ = leveldb.OpenFile(dbName, nil)
	}
}

func setupTestDataAccess() DataAccess {
	return &DataAccessImpl{
		db:          db,
		transaction: new(leveldb.Batch),
		cache:       &fakeCache{},
	}
}

func removeDir(dirName string) {
	dirPath, _ := filepath.Abs(dirName)
	os.RemoveAll(dirPath)
}

func teardownTestDataAccess() {
	db.Close()
	removeDir(dbName)
}

func TestMain(m *testing.M) {
	initializeLevelDB()
	result := m.Run()
	teardownTestDataAccess()
	os.Exit(result)
}

func TestBeginShouldErrorWhenAlreadyInTransaction(t *testing.T) {
	da := setupTestDataAccess()

	err := da.Begin()
	if nil != err {
		t.Errorf("Error first time Begin should success")
	}

	err = da.Begin()
	if nil == err {
		t.Errorf("Error second time Begin should return error")
	}
}

func TestCommitUnlockInUse(t *testing.T) {
	da := setupTestDataAccess()

	_ = da.Begin()
	da.Commit()

	err := da.Begin()
	if nil != err {
		t.Errorf("Error Commit didn't reset variable inUse")
	}
}

func TestCommitResetTransaction(t *testing.T) {
	da := setupTestDataAccess()

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte{'a'},
		[]byte{'b'},
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	da.Commit()

	actual := da.DumpTx()

	if 0 != len(actual) {
		t.Errorf("Error commit didn't reset transaction")
	}
}

func TestCommitWriteToDB(t *testing.T) {
	da := setupTestDataAccess()

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte{'a'},
		[]byte{'b'},
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	da.Commit()

	actual, _ := da.Get(fixture.key)
	if !isSameByteSlice(actual, fixture.value) {
		t.Errorf("Error commit didn't write to db, expect %v but get %v",
			fixture.value, actual)
	}
}

func TestPutActionCached(t *testing.T) {
	da := setupTestDataAccess()

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte{'a'},
		[]byte{'b'},
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	if !setCalled {
		t.Errorf("Error put action didn't cache data")
	}
}

func TestDeleteActionCached(t *testing.T) {
	da := setupTestDataAccess()

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
	if !setCalled {
		t.Errorf("Error put action didn't cache data")
	}
}

func TestCommitClearsCache(t *testing.T) {
	da := setupTestDataAccess()

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte{'a'},
		[]byte{'b'},
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	da.Commit()
	if !clearCalled {
		t.Errorf("Error clear action didn't reset cache")
	}
}

func TestGetActionReadsFromCache(t *testing.T) {
	da := setupTestDataAccess()

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

	if !getCalled {
		t.Errorf("Error clear action didn't reset cache")
	}

	if !isSameByteSlice(actual, fixture.value) {
		t.Errorf("Error cached value, expect %v but get %v", fixture.value, actual)
	}
}

func TestGetActionReadDBIfNotInCache(t *testing.T) {
	da := setupTestDataAccess()

	fixture := struct {
		key   []byte
		value []byte
	}{
		[]byte("random"),
		[]byte{'a', 'b', 'c'},
	}

	_ = da.Begin()
	da.Put(fixture.key, fixture.value)
	da.Commit()
	actual, _ := da.Get(fixture.key)

	if !isSameByteSlice(actual, fixture.value) {
		t.Errorf("Error get db value, expect %v but get %v", fixture.value, actual)
	}
}
