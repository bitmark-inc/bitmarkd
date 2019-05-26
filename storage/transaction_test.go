package storage

import (
	"os"
	"testing"

	"github.com/bitmark-inc/bitmarkd/storage/mocks"
	"github.com/bitmark-inc/logger"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

const (
	testingDirName = "testing"
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

func removeFiles() {
	os.RemoveAll(testingDirName)
}

func teardownTestLogger() {
	removeFiles()
}

func newTestMockDataAccess(t *testing.T) (*mocks.MockDataAccess, *gomock.Controller) {
	ctl := gomock.NewController(t)

	return mocks.NewMockDataAccess(ctl), ctl
}

func setupTestTransaction(t *testing.T) (Transaction, *mocks.MockDataAccess, *gomock.Controller) {
	mock, ctl := newTestMockDataAccess(t)

	trx := newTransaction([]DataAccess{mock})
	return trx, mock, ctl
}

func TestBegin(t *testing.T) {
	tx, mock, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mock.EXPECT().Begin().Return(nil).Times(1)
	mock.EXPECT().InUse().Return(false).Times(1)
	mock.EXPECT().InUse().Return(true).Times(1)

	err := tx.Begin()
	assert.Equal(t, nil, err, "first time Begin should not return any error")

	err = tx.Begin()
	assert.NotEqual(t, nil, err, "second time Begin should return error")
}

// this is ugly, because it uses unexported method, so general gomock cannot be used
type testHandleMock struct {
	Handle
	PutCalled    bool
	PutNCalled   bool
	RemoveCalled bool
	GetCalled    bool
}

func (m *testHandleMock) Put(key []byte, value []byte, dummy []byte) {}
func (m *testHandleMock) put(key []byte, value []byte, dummy []byte) { m.PutCalled = true }
func (m *testHandleMock) PutN(key []byte, value uint64)              {}
func (m *testHandleMock) putN(key []byte, value uint64)              { m.PutNCalled = true }
func (m *testHandleMock) Delete(key []byte)                          {}
func (m *testHandleMock) remove(key []byte)                          { m.RemoveCalled = true }
func (m *testHandleMock) Get(key []byte) []byte {
	m.GetCalled = true
	return []byte{}
}
func (m *testHandleMock) GetN(key []byte) (uint64, bool) {
	m.GetCalled = true
	return uint64(0), true
}
func (m *testHandleMock) GetNB(key []byte) (uint64, []byte) {
	m.GetCalled = true
	return uint64(0), []byte{}
}
func (m *testHandleMock) Has(key []byte) bool { return true }
func (m *testHandleMock) Begin()              {}
func (m *testHandleMock) Commit() error       { return nil }

func newTestHandleMock() *testHandleMock {
	return &testHandleMock{
		PutCalled:    false,
		PutNCalled:   false,
		RemoveCalled: false,
		GetCalled:    false,
	}
}

func TestPut(t *testing.T) {
	tx, mockDA, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mockDA.EXPECT().Begin().Times(1)
	mockDA.EXPECT().InUse().Return(false).Times(1)
	myMock := newTestHandleMock()

	_ = tx.Begin()
	err := tx.Put(myMock, []byte{}, []byte{}, []byte{})

	assert.Equal(t, true, myMock.PutCalled, "internal method put is not called")
	assert.Equal(t, nil, err, err)
}

func TestPutN(t *testing.T) {
	tx, mockDA, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mockDA.EXPECT().Begin().Times(1)
	mockDA.EXPECT().InUse().Return(false).Times(1)
	myMock := newTestHandleMock()

	_ = tx.Begin()

	tx.PutN(myMock, []byte{}, uint64(0))

	assert.Equal(t, true, myMock.PutNCalled, "internal method putN not called")
}

func TestDelete(t *testing.T) {
	tx, mockDA, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mockDA.EXPECT().Begin().Times(1)
	mockDA.EXPECT().InUse().Return(false).Times(1)
	myMock := newTestHandleMock()

	_ = tx.Begin()
	err := tx.Delete(myMock, []byte{})

	assert.Equal(t, true, myMock.RemoveCalled, "internal method remove not called")
	assert.Equal(t, nil, err, err)
}

func TestGet(t *testing.T) {
	tx, mockDA, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mockDA.EXPECT().Begin().Times(1)
	mockDA.EXPECT().InUse().Return(false).Times(1)
	myMock := newTestHandleMock()

	_ = tx.Begin()
	_, err := tx.Get(myMock, []byte{})

	assert.Equal(t, true, myMock.GetCalled, "internal method get not called")
	assert.Equal(t, nil, err, err)
}

func TestGetN(t *testing.T) {
	tx, mockDA, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mockDA.EXPECT().Begin().Times(1)
	mockDA.EXPECT().InUse().Return(false).Times(1)
	myMock := newTestHandleMock()

	_ = tx.Begin()
	_, _, err := tx.GetN(myMock, []byte{})

	assert.Equal(t, true, myMock.GetCalled, "internal method get is not called")
	assert.Equal(t, nil, err, err)
}

func TestGetNB(t *testing.T) {
	setupTestLogger()
	defer teardownTestLogger()

	tx, mockDA, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mockDA.EXPECT().Begin().Times(1)
	mockDA.EXPECT().InUse().Return(false).Times(1)
	myMock := newTestHandleMock()

	_ = tx.Begin()
	_, _, err := tx.GetNB(myMock, []byte{})

	assert.Equal(t, true, myMock.GetCalled, "internal method get is not called")
	assert.Equal(t, nil, err, err)
}

func TestCommit(t *testing.T) {
	tx, mock, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mock.EXPECT().Commit().Return(nil).Times(1)
	mock.EXPECT().Begin().Times(2)
	mock.EXPECT().InUse().Return(false).Times(2)

	_ = tx.Begin()
	_ = tx.Commit()

	err := tx.Begin()
	assert.Equal(t, nil, err, "did not release lock")
}

func TestIsNilPtr(t *testing.T) {
	err := isNilPtr(nil)
	assert.NotEqual(t, nil, err, "cannot check nil pointer")

	str := struct{}{}
	err = isNilPtr(&str)
	assert.Equal(t, nil, err, "cannot check non-nil pointer")
}

func TestAbortCallsDataAccessAbort(t *testing.T) {
	tx, mock, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mock.EXPECT().Begin().Times(1)
	mock.EXPECT().InUse().Return(false).Times(1)
	mock.EXPECT().Abort().Times(1)

	_ = tx.Begin()
	tx.Abort()
}

func TestTxInUse(t *testing.T) {
	tx, mock, ctl := setupTestTransaction(t)
	defer ctl.Finish()

	mock.EXPECT().InUse().Return(false).Times(1)
	mock.EXPECT().InUse().Return(true).Times(1)

	inUse := tx.InUse()
	assert.Equal(t, false, inUse, "inUse incorrect")

	inUse = tx.InUse()
	assert.Equal(t, true, inUse, "inUse incorrect")
}
