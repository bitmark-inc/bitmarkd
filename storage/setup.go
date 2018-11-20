// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
	ldb_opt "github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

// exported storage pools
//
// note all must be exported (i.e. initial capital) or initialisation will panic
type pools struct {
	Blocks            *PoolHandle `prefix:"B" database:"blocks"`
	BlockOwnerPayment *PoolHandle `prefix:"H" database:"index"`
	BlockOwnerTxIndex *PoolHandle `prefix:"I" database:"index"`
	Assets            *PoolNB     `prefix:"A" database:"index"`
	Transactions      *PoolNB     `prefix:"T" database:"index"`
	OwnerNextCount    *PoolHandle `prefix:"N" database:"index"`
	OwnerList         *PoolHandle `prefix:"L" database:"index"`
	OwnerTxIndex      *PoolHandle `prefix:"D" database:"index"`
	OwnerData         *PoolHandle `prefix:"O" database:"index"`
	Shares            *PoolHandle `prefix:"F" database:"index"`
	ShareQuantity     *PoolHandle `prefix:"Q" database:"index"`
	TestData          *PoolHandle `prefix:"Z" database:"index"`
}

// the instance
var Pool pools

// for database version
var versionKey = []byte{0x00, 'V', 'E', 'R', 'S', 'I', 'O', 'N'}

const (
	currentVersion = 0x200
)

// holds the database handle
var poolData struct {
	sync.RWMutex
	dbBlocks *leveldb.DB
	dbIndex  *leveldb.DB
}

const (
	ReadOnly  = true
	ReadWrite = false
)

// open up the database connection
//
// this must be called before any pool is accessed
func Initialise(database string, readOnly bool) (bool, error) {
	poolData.Lock()
	defer poolData.Unlock()

	ok := false
	mustReindex := false

	if nil != poolData.dbBlocks {
		return mustReindex, fault.ErrAlreadyInitialised
	}

	defer func() {
		if !ok {
			dbClose()
		}
	}()

	blocksDatabase := database + "-blocks.leveldb"
	indexDatabase := database + "-index.leveldb"

	db, blocksVersion, err := getDB(blocksDatabase, readOnly)
	if nil != err {
		return mustReindex, err
	}
	poolData.dbBlocks = db

	// ensure no database downgrade
	if blocksVersion > currentVersion {
		logger.Criticalf("block database version: %d > current version: %d", blocksVersion, currentVersion)
		return mustReindex, fmt.Errorf("block database version: %d > current version: %d", blocksVersion, currentVersion)
	}

	db, indexVersion, err := getDB(indexDatabase, readOnly)
	if nil != err {
		return mustReindex, err
	}
	poolData.dbIndex = db

	// ensure no database downgrade
	if indexVersion > currentVersion {
		logger.Criticalf("index database version: %d > current version: %d", indexVersion, currentVersion)
		return mustReindex, fmt.Errorf("index database version: %d > current version: %d", indexVersion, currentVersion)
	}

	// prevent readOnly from modifying the database
	if readOnly && (blocksVersion != currentVersion || indexVersion != currentVersion) {
		logger.Criticalf("database is inconsistent: blocks: %d  index: %d  current: %d", blocksVersion, indexVersion, currentVersion)
		return mustReindex, fmt.Errorf("database is inconsistent: blocks: %d  index: %d  current: %d", blocksVersion, indexVersion, currentVersion)
	}

	if 0 < blocksVersion && blocksVersion < currentVersion {

		// fail if block database is too old
		// this will be replaced by the appropriate migration code
		// if the format of blocks needs to be changed in the future

		logger.Criticalf("no migration for block database version: %d", blocksVersion)
		logger.Criticalf("block database version: %d < current version: %d", blocksVersion, currentVersion)
		return mustReindex, fmt.Errorf("block database version: %d < current version: %d", blocksVersion, currentVersion)

	} else if 0 == blocksVersion {

		// database was empty so tag as current version
		err = putVersion(poolData.dbBlocks, currentVersion)
		if err != nil {
			return mustReindex, err
		}
	}

	// see if index need to be created or deleted and re-created
	if mustReindex || indexVersion < currentVersion {

		mustReindex = true

		// close out current index
		poolData.dbIndex.Close()
		poolData.dbIndex = nil

		logger.Criticalf("drop index database: %s", indexDatabase)

		// erase the index completely
		err = os.RemoveAll(indexDatabase)
		if nil != err {
			return mustReindex, err
		}

		// generate an empty index database
		poolData.dbIndex, _, err = getDB(indexDatabase, readOnly)
		if nil != err {
			return mustReindex, err
		}

	}

	// this will be a struct type
	poolType := reflect.TypeOf(Pool)

	// get write access by using pointer + Elem()
	poolValue := reflect.ValueOf(&Pool).Elem()

	// scan each field
	for i := 0; i < poolType.NumField(); i += 1 {

		fieldInfo := poolType.Field(i)

		prefixTag := fieldInfo.Tag.Get("prefix")
		if 1 != len(prefixTag) {
			return mustReindex, fmt.Errorf("pool: %v  has invalid prefix: %q", fieldInfo, prefixTag)
		}

		prefix := prefixTag[0]
		limit := []byte(nil)
		if prefix < 255 {
			limit = []byte{prefix + 1}
		}

		db := poolData.dbIndex
		switch dbName := fieldInfo.Tag.Get("database"); dbName {
		case "blocks":
			db = poolData.dbBlocks
		case "index":
			db = poolData.dbIndex
		default:
			return mustReindex, fmt.Errorf("pool: %v  has invalid database: %q", fieldInfo, dbName)
		}

		p := &PoolHandle{
			prefix:   prefix,
			limit:    limit,
			database: db,
		}

		if poolValue.Field(i).Type() == reflect.TypeOf((*PoolNB)(nil)) {
			pNB := &PoolNB{
				pool: p,
			}
			newNB := reflect.ValueOf(pNB)
			poolValue.Field(i).Set(newNB)
		} else {
			newPool := reflect.ValueOf(p)
			poolValue.Field(i).Set(newPool)
		}

	}

	ok = true // prevent db close
	return mustReindex, nil
}

func dbClose() {
	if nil != poolData.dbIndex {
		poolData.dbIndex.Close()
		poolData.dbIndex = nil
	}
	if nil != poolData.dbBlocks {
		poolData.dbBlocks.Close()
		poolData.dbBlocks = nil
	}
}

// close the database connection
func Finalise() {
	poolData.Lock()
	dbClose()
	poolData.Unlock()
}

// called at the end of reindex
func ReindexDone() error {
	poolData.Lock()
	defer poolData.Unlock()
	return putVersion(poolData.dbIndex, currentVersion)
}

// return:
//   databse handle
//   version number
func getDB(name string, readOnly bool) (*leveldb.DB, int, error) {

	opt := &ldb_opt.Options{
		ErrorIfExist:   false,
		ErrorIfMissing: readOnly,
		ReadOnly:       readOnly,
	}

	db, err := leveldb.OpenFile(name, opt)
	if nil != err {
		return nil, 0, err
	}

	versionValue, err := db.Get(versionKey, nil)
	if leveldb.ErrNotFound == err {
		return db, 0, nil
	} else if nil != err {
		db.Close()
		return nil, 0, err
	}

	if 4 != len(versionValue) {
		db.Close()
		return nil, 0, fmt.Errorf("incompatible database version length: expected: %d  actual: %d", 4, len(versionValue))
	}

	version := int(binary.BigEndian.Uint32(versionValue))
	return db, version, nil
}

func putVersion(db *leveldb.DB, version int) error {

	currentVersion := make([]byte, 4)
	binary.BigEndian.PutUint32(currentVersion, uint32(version))

	return db.Put(versionKey, currentVersion, nil)
}
