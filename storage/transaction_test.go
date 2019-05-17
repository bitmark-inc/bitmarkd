package storage

import (
	"os"
	"testing"

	"github.com/bitmark-inc/logger"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	ldb_util "github.com/syndtr/goleveldb/leveldb/util"
)

const (
	testingDirName = "testing"
)

var (
	isPutCalled    = false
	isDeleteCalled = false
	isGetCalled    = false
	isCommitCalled = false
	ph             = &PoolHandle{
		prefix:     'a',
		limit:      []byte{2},
		dataAccess: &fakeDataAccess{},
	}
)

type fakeDataAccess struct{}

func (f *fakeDataAccess) Begin() error       { return nil }
func (f *fakeDataAccess) Put([]byte, []byte) { isPutCalled = true }
func (f *fakeDataAccess) Delete([]byte)      { isDeleteCalled = true }
func (f *fakeDataAccess) Commit() error {
	isCommitCalled = true
	return nil
}
func (f *fakeDataAccess) Get([]byte) ([]byte, error) {
	isGetCalled = true
	return []byte{'1', '2', '3', '4', '5', '6', '7', '8', '9'}, nil // to pass getNB
}
func (f *fakeDataAccess) Iterator(*ldb_util.Range) iterator.Iterator {
	return &fakeIterator{}
}
func (f *fakeDataAccess) DumpTx() []byte           { return []byte{} }
func (f *fakeDataAccess) Has([]byte) (bool, error) { return true, nil }

type fakeIterator struct{}

func (f *fakeIterator) Valid() bool                   { return true }
func (f *fakeIterator) Error() error                  { return nil }
func (f *fakeIterator) Key() []byte                   { return []byte{} }
func (f *fakeIterator) Value() []byte                 { return []byte{} }
func (f *fakeIterator) First() bool                   { return true }
func (f *fakeIterator) Last() bool                    { return true }
func (f *fakeIterator) Seek([]byte) bool              { return true }
func (f *fakeIterator) Next() bool                    { return true }
func (f *fakeIterator) Prev() bool                    { return true }
func (f *fakeIterator) Release()                      {}
func (f *fakeIterator) SetReleaser(ldb_util.Releaser) {}

func setupTestLogger() {
	removeFiles()
	os.Mkdir(testingDirName, 0700)

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

func removeFiles() {
	os.RemoveAll(testingDirName)
}

func teardownTestLogger() {
	removeFiles()
}

func setupTestTransaction() Transaction {
	return &TransactionImpl{
		inUse: false,
	}
}

func TestBegin(t *testing.T) {
	tx := setupTestTransaction()

	err := tx.Begin()
	if nil != err {
		t.Errorf("Error first time Begin, should not return any error")
	}

	err = tx.Begin()
	if nil == err {
		t.Errorf("Error second time Begin, should return error")
	}
}

func TestPut(t *testing.T) {
	tx := setupTestTransaction()
	_ = tx.Begin()
	tx.Put(ph, []byte{}, []byte{})

	if !isPutCalled {
		t.Errorf("Error put is not called")
	}
}

func TestDelete(t *testing.T) {
	tx := setupTestTransaction()
	_ = tx.Begin()
	tx.Delete(ph, []byte{})

	if !isDeleteCalled {
		t.Errorf("Error remove is not called")
	}
}

func TestGet(t *testing.T) {
	tx := setupTestTransaction()
	_ = tx.Begin()
	tx.Get(ph, []byte{})

	if !isGetCalled {
		t.Errorf("Error get is not called")
	}
}

func TestGetN(t *testing.T) {
	tx := setupTestTransaction()
	_ = tx.Begin()
	isGetCalled = false
	tx.Get(ph, []byte{})

	if !isGetCalled {
		t.Errorf("Error get is not called")
	}
}

func TestGetNB(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()
	tx := setupTestTransaction()
	_ = tx.Begin()
	isGetCalled = false
	tx.GetNB(ph, []byte{})

	if !isGetCalled {
		t.Errorf("Error get is not called")
	}
}

func TestPutN(t *testing.T) {
	tx := setupTestTransaction()
	_ = tx.Begin()
	isPutCalled = false
	tx.PutN(ph, []byte{}, uint64(0))

	if !isPutCalled {
		t.Errorf("Error putN is not called")
	}
}

func TestCommit(t *testing.T) {
	tx := setupTestTransaction()
	_ = tx.Begin()
	_ = tx.Begin()
	err := tx.Commit(ph)

	if !isCommitCalled {
		t.Errorf("Error Commit not call member function Commit")
	}

	if nil != err {
		t.Error("Error Commit didn't reset inUse")
	}

	err = tx.Begin()
	if nil != err {
		t.Error("Erro Commit didn't refresh lock")
	}
}
