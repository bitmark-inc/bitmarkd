package storage

import (
	"fmt"
	"sync"
)

// Transaction RDBS transaction
type Transaction interface {
	Abort()
	Begin() error
	Commit() error
	Delete(Handle, []byte)
	Get(Handle, []byte) []byte
	GetN(Handle, []byte) (uint64, bool)
	GetNB(Handle, []byte) (uint64, []byte)
	Has(Handle, []byte) bool
	InUse() bool
	Put(Handle, []byte, []byte, []byte)
	PutN(Handle, []byte, uint64)
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
		return fmt.Errorf("transaction already in use")
	}

	for _, access := range t.dataAccess {
		access.Begin()
	}

	return nil
}

func (t *TransactionImpl) Put(
	h Handle,
	key []byte,
	value []byte,
	additional []byte,
) {
	h.put(key, value, additional)
}

func (t *TransactionImpl) PutN(h Handle, key []byte, value uint64) {
	h.putN(key, value)
}

func (t *TransactionImpl) Delete(h Handle, key []byte) {
	h.remove(key)
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

func (t *TransactionImpl) Get(h Handle, key []byte) []byte {
	return h.Get(key)
}

func (t *TransactionImpl) GetN(h Handle, key []byte) (uint64, bool) {
	num, found := h.GetN(key)
	return num, found
}

func (t *TransactionImpl) GetNB(h Handle, key []byte) (uint64, []byte) {
	num, buffer := h.GetNB(key)
	return num, buffer
}

func (t *TransactionImpl) Abort() {
	for _, da := range t.dataAccess {
		da.Abort()
	}
}

func (t *TransactionImpl) Has(h Handle, key []byte) bool {
	return h.Has(key)
}
