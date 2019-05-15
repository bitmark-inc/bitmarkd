package storage

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	ldb_util "github.com/syndtr/goleveldb/leveldb/util"
)

// Transaction RDBS transaction
type DataAccess interface {
	Begin()
	Put([]byte, []byte)
	Delete([]byte)
	Write() error
	Get([]byte) ([]byte, error)
	Iterator(*ldb_util.Range) iterator.Iterator
	Has([]byte) (bool, error)
}

type DataAccessImpl struct {
	db          *leveldb.DB
	transaction *leveldb.Batch
}

func newDB(db *leveldb.DB) DataAccess {
	return &DataAccessImpl{
		db:          db,
		transaction: new(leveldb.Batch),
	}
}

func (d *DataAccessImpl) Begin() {
	d.transaction.Reset()
}

func (d *DataAccessImpl) Put(key []byte, value []byte) {
	d.transaction.Put(key, value)
}

func (d *DataAccessImpl) Delete(key []byte) {
	d.transaction.Delete(key)
}

func (d *DataAccessImpl) Write() error {
	err := d.db.Write(d.transaction, nil)
	d.Begin()
	return err
}

func (d *DataAccessImpl) Get(key []byte) ([]byte, error) {
	val, err := d.db.Get(key, nil)
	return val, err
}

func (d *DataAccessImpl) Iterator(searchRange *ldb_util.Range) iterator.Iterator {
	return d.db.NewIterator(searchRange, nil)
}

func (d *DataAccessImpl) Has(key []byte) (bool, error) {
	return d.db.Has(key, nil)
}
