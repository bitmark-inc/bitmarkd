package storage

import (
	"fmt"
	"sync"
)

// Transaction RDBS transaction
type Transaction interface {
	Begin() error
	Put(*PoolHandle, []byte, []byte)
	PutN(*PoolHandle, []byte, uint64)
	Delete(*PoolHandle, []byte)
	Get(*PoolHandle, []byte) []byte
	GetN(*PoolHandle, []byte) (uint64, bool)
	GetNB(*PoolHandle, []byte) (uint64, []byte)
	Commit(*PoolHandle) error
}

type TransactionImpl struct {
	sync.Mutex
	inUse bool
}

func newTransaction() Transaction {
	return &TransactionImpl{
		inUse: false,
	}
}

func (d *TransactionImpl) Begin() error {
	if d.inUse {
		return fmt.Errorf("Error, transaction already in use")
	}

	d.Lock()
	d.inUse = true
	d.Unlock()

	return nil
}

func (d *TransactionImpl) Put(handle *PoolHandle, key []byte, value []byte) {
	handle.put(key, value)
}

func (d *TransactionImpl) PutN(handle *PoolHandle, key []byte, value uint64) {
	handle.putN(key, value)
}

func (d *TransactionImpl) Delete(handle *PoolHandle, key []byte) {
	handle.remove(key)
}

func (d *TransactionImpl) Commit(handle *PoolHandle) error {
	d.Lock()
	defer d.Unlock()

	d.inUse = false
	handle.Commit()

	return nil
}

func (d *TransactionImpl) Get(ph *PoolHandle, key []byte) []byte {
	return ph.Get(key)
}

func (d *TransactionImpl) GetN(ph *PoolHandle, key []byte) (uint64, bool) {
	return ph.getN(key)
}

func (d *TransactionImpl) GetNB(ph *PoolHandle, key []byte) (uint64, []byte) {
	return ph.getNB(key)
}
