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
	Begin() error
	Put(*PoolHandle, []byte, []byte) error
	PutN(*PoolHandle, []byte, uint64) error
	Delete(*PoolHandle, []byte) error
	Get(*PoolHandle, []byte) ([]byte, error)
	GetN(*PoolHandle, []byte) (uint64, bool, error)
	GetNB(*PoolHandle, []byte) (uint64, []byte, error)
	Commit() error
}

type TransactionImpl struct {
	sync.Mutex
	inUse      bool
	dataAccess []DataAccess
}

func newTransaction(access []DataAccess) Transaction {
	return &TransactionImpl{
		inUse:      false,
		dataAccess: access,
	}
}

func isNilPtr(ptr interface{}) error {
	if nil == ptr {
		return fmt.Errorf(ErrHandleNil)
	}
	return nil
}

func (d *TransactionImpl) Begin() error {
	if d.inUse {
		return fmt.Errorf("Error, transaction already in use")
	}

	d.Lock()
	d.inUse = true
	d.Unlock()

	for _, access := range d.dataAccess {
		access.Begin()
	}

	return nil
}

func (d *TransactionImpl) Put(ph *PoolHandle, key []byte, value []byte) error {
	if nil == ph {
		return fmt.Errorf(ErrHandleNil)
	}

	ph.put(key, value)
	return nil
}

func (d *TransactionImpl) PutN(ph *PoolHandle, key []byte, value uint64) error {
	err := isNilPtr(ph)
	if nil != err {
		return err
	}

	ph.putN(key, value)
	return nil
}

func (d *TransactionImpl) Delete(ph *PoolHandle, key []byte) error {
	err := isNilPtr(ph)
	if nil != err {
		return err
	}

	ph.remove(key)
	return nil
}

func (d *TransactionImpl) Commit() error {
	d.Lock()
	d.inUse = false
	defer d.Unlock()

	for _, access := range d.dataAccess {
		err := access.Commit()
		if nil != err {
			return err
		}
	}
	return nil
}

func (d *TransactionImpl) Get(ph *PoolHandle, key []byte) ([]byte, error) {
	err := isNilPtr(ph)
	if nil != err {
		return []byte{}, err
	}

	return ph.Get(key), nil
}

func (d *TransactionImpl) GetN(ph *PoolHandle, key []byte) (uint64, bool, error) {
	err := isNilPtr(ph)
	if nil != err {
		return uint64(0), false, err
	}

	num, found := ph.getN(key)
	return num, found, nil
}

func (d *TransactionImpl) GetNB(ph *PoolHandle, key []byte) (uint64, []byte, error) {
	err := isNilPtr(ph)
	if nil != err {
		return uint64(0), []byte{}, err
	}

	num, buffer := ph.getNB(key)
	return num, buffer, nil
}
