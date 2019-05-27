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
	Abort()
	Begin() error
	Commit() error
	Delete([]byte)
	DumpTx() []byte
	Get([]byte) ([]byte, error)
	Has([]byte) (bool, error)
	InUse() bool
	Iterator(*ldb_util.Range) iterator.Iterator
	Put([]byte, []byte)
}

type DataAccessImpl struct {
	sync.Mutex
	inUse bool
	db    *leveldb.DB
	batch *leveldb.Batch
	cache Cache
}

func newDA(db *leveldb.DB, trx *leveldb.Batch, cache Cache) DataAccess {
	return &DataAccessImpl{
		inUse: false,
		db:    db,
		batch: trx,
		cache: cache,
	}
}

func (d *DataAccessImpl) Begin() error {
	d.Lock()
	defer d.Unlock()

	if d.inUse {
		return fmt.Errorf("Error, batch already in use")
	}

	d.inUse = true
	return nil
}

func (d *DataAccessImpl) Put(key []byte, value []byte) {
	d.cache.Set(dbPut, string(key), value)
	d.batch.Put(key, value)
}

func (d *DataAccessImpl) Delete(key []byte) {
	d.cache.Set(dbDelete, string(key), []byte{})
	d.batch.Delete(key)
}

func (d *DataAccessImpl) Commit() error {
	err := d.db.Write(d.batch, nil)
	if nil != err {
		return err
	}
	return nil
}

func (d *DataAccessImpl) DumpTx() []byte {
	return d.batch.Dump()
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
	_, found := d.getFromCache(key)
	if found {
		return true, nil
	}
	return d.db.Has(key, nil)
}

func (d *DataAccessImpl) InUse() bool {
	return d.inUse
}

func (d *DataAccessImpl) Abort() {
	d.Lock()
	defer d.Unlock()

	d.batch.Reset()
	d.cache.Clear()
	d.inUse = false
}
