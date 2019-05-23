package storage

import (
	"fmt"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	ldb_util "github.com/syndtr/goleveldb/leveldb/util"
)

// for Database
type DataAccess interface {
	Begin() error
	Put([]byte, []byte)
	Delete([]byte)
	Commit() error
	Get([]byte) ([]byte, error)
	Iterator(*ldb_util.Range) iterator.Iterator
	DumpTx() []byte
	Has([]byte) (bool, error)
}

type DataAccessImpl struct {
	sync.Mutex
	inUse       bool
	db          *leveldb.DB
	transaction *leveldb.Batch
	cache       Cache
}

func newDA(db *leveldb.DB, trx *leveldb.Batch, cache Cache) DataAccess {
	return &DataAccessImpl{
		inUse:       false,
		db:          db,
		transaction: trx,
		cache:       cache,
	}
}

func (d *DataAccessImpl) Begin() error {
	d.Lock()
	defer d.Unlock()

	if d.inUse {
		return fmt.Errorf("Error, transaction already in use")
	}

	d.inUse = true
	return nil
}

func (d *DataAccessImpl) Put(key []byte, value []byte) {
	d.cache.Set(dbPut, string(key), value)
	d.transaction.Put(key, value)
}

func (d *DataAccessImpl) Delete(key []byte) {
	d.transaction.Delete(key)
}

func (d *DataAccessImpl) Commit() error {
	err := d.db.Write(d.transaction, nil)
	d.transaction.Reset()
	d.cache.Clear()
	d.Lock()
	defer d.Unlock()

	d.inUse = false
	return err
}

func (d *DataAccessImpl) DumpTx() []byte {
	return d.transaction.Dump()
}

func (d *DataAccessImpl) Get(key []byte) ([]byte, error) {
	val, found := d.getFromCache(key)
	if found {
		return val, nil
	}
	return d.getFromDB(key)
}

func (d *DataAccessImpl) getFromCache(key []byte) ([]byte, bool) {
	return d.cache.Get(string(key))
}

func (d *DataAccessImpl) getFromDB(key []byte) ([]byte, error) {
	return d.db.Get(key, nil)
}

func (d *DataAccessImpl) Iterator(searchRange *ldb_util.Range) iterator.Iterator {
	return d.db.NewIterator(searchRange, nil)
}

func (d *DataAccessImpl) Has(key []byte) (bool, error) {
	return d.db.Has(key, nil)
}
