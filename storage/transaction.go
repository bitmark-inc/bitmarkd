package storage

import (
	"fmt"
	"sync"
)

const (
	ErrHandleNil = "Error handle is nil"
)

// Transaction RDBS transaction
type Transaction interface {
	Abort()
	Begin() error
	Commit() error
	Delete(Handle, []byte) error
	Get(Handle, []byte) ([]byte, error)
	GetN(Handle, []byte) (uint64, bool, error)
	GetNB(Handle, []byte) (uint64, []byte, error)
	Has(Handle, []byte) bool
	InUse() bool
	Put(Handle, []byte, []byte, []byte) error
	PutN(Handle, []byte, uint64) error
}

type TransactionImpl struct {
	sync.Mutex
	dataAccess []DataAccess
}

func newTransaction(access []DataAccess) Transaction {
	return &TransactionImpl{
		dataAccess: access,
	}
}

func isNilPtr(ptr interface{}) error {
	if nil == ptr {
		return fmt.Errorf(ErrHandleNil)
	}
	return nil
}

func (t *TransactionImpl) InUse() bool {
	for _, da := range t.dataAccess {
		if da.InUse() {
			return true
		}
	}
	return false
}

func (t *TransactionImpl) Begin() error {
	if t.InUse() {
		return fmt.Errorf("Error, transaction already in use")
	}

	for _, access := range t.dataAccess {
		access.Begin()
	}

	return nil
}

func (t *TransactionImpl) Put(h Handle, key []byte, value []byte, dummy []byte) error {
	if nil == h {
		return fmt.Errorf(ErrHandleNil)
	}

	h.put(key, value, []byte{})
	return nil
}

func (t *TransactionImpl) PutN(h Handle, key []byte, value uint64) error {
	err := isNilPtr(h)
	if nil != err {
		return err
	}

	h.putN(key, value)
	return nil
}

func (t *TransactionImpl) Delete(h Handle, key []byte) error {
	err := isNilPtr(h)
	if nil != err {
		return err
	}

	h.remove(key)
	return nil
}

func (t *TransactionImpl) Commit() error {
	for _, access := range t.dataAccess {
		err := access.Commit()
		if nil != err {
			return err
		}
	}
	t.Abort()
	return nil
}

func (t *TransactionImpl) Get(h Handle, key []byte) ([]byte, error) {
	err := isNilPtr(h)
	if nil != err {
		return []byte{}, err
	}

	return h.Get(key), nil
}

func (t *TransactionImpl) GetN(h Handle, key []byte) (uint64, bool, error) {
	err := isNilPtr(h)
	if nil != err {
		return uint64(0), false, err
	}

	num, found := h.GetN(key)
	return num, found, nil
}

func (t *TransactionImpl) GetNB(h Handle, key []byte) (uint64, []byte, error) {
	err := isNilPtr(h)
	if nil != err {
		return uint64(0), []byte{}, err
	}

	num, buffer := h.GetNB(key)
	return num, buffer, nil
}

func (t *TransactionImpl) Abort() {
	for _, da := range t.dataAccess {
		da.Abort()
	}
}

func (t *TransactionImpl) Has(h Handle, key []byte) bool {
	return h.Has(key)
}
