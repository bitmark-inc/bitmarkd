// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
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
	BlockHeaderHash   *PoolHandle `prefix:"2" database:"blocks"`
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

// Pool - the set of exported pools
var Pool pools

// for database version
var versionKey = []byte{0x00, 'V', 'E', 'R', 'S', 'I', 'O', 'N'}

const (
	currentBlockDBVersion = 0x301
	currentIndexDBVersion = 0x200
	ErrEmptyTransaction   = "Empty Transaction"
)

// holds the database handle
var poolData struct {
	sync.RWMutex
	dbBlocks  *leveldb.DB
	dbIndex   *leveldb.DB
	trx       Transaction
	blocksTrx *leveldb.Batch
	indexTrx  *leveldb.Batch
	cache     Cache
}

// pool access modes
const (
	ReadOnly  = true
	ReadWrite = false
)

// Initialise - open up the database connection
//
// this must be called before any pool is accessed
func Initialise(database string, readOnly bool) (bool, bool, error) {
	poolData.Lock()
	defer poolData.Unlock()

	ok := false
	mustMigrate := false
	mustReindex := false

	if nil != poolData.dbBlocks {
		return mustMigrate, mustReindex, fault.ErrAlreadyInitialised
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
		return mustMigrate, mustReindex, err
	}
	poolData.dbBlocks = db

	// ensure no database downgrade
	if blocksVersion > currentBlockDBVersion {
		logger.Criticalf("block database version: %d > current version: %d", blocksVersion, currentBlockDBVersion)
		return mustMigrate, mustReindex, fmt.Errorf("block database version: %d > current version: %d", blocksVersion, currentBlockDBVersion)
	}

	db, indexVersion, err := getDB(indexDatabase, readOnly)
	if nil != err {
		return mustMigrate, mustReindex, err
	}
	poolData.dbIndex = db

	// ensure no database downgrade
	if indexVersion > currentIndexDBVersion {
		logger.Criticalf("index database version: %d > current version: %d", indexVersion, currentIndexDBVersion)
		return mustMigrate, mustReindex, fmt.Errorf("index database version: %d > current version: %d", indexVersion, currentIndexDBVersion)
	}

	// prevent readOnly from modifying the database
	if readOnly && (blocksVersion != currentBlockDBVersion || indexVersion != currentIndexDBVersion) {
		logger.Criticalf("database is inconsistent: blocks: %d  index: %d  current: %d & %d", blocksVersion, indexVersion, currentBlockDBVersion, currentIndexDBVersion)
		return mustMigrate, mustReindex, fmt.Errorf("database is inconsistent: blocks: %d  index: %d  current: %d & %d", blocksVersion, indexVersion, currentBlockDBVersion, currentIndexDBVersion)
	}

	if 0 < blocksVersion && blocksVersion < currentBlockDBVersion {

		mustMigrate = true

		logger.Criticalf("block database version: %d < current version: %d", blocksVersion, currentBlockDBVersion)

	} else if 0 == blocksVersion {

		// database was empty so tag as current version
		err = putVersion(poolData.dbBlocks, currentBlockDBVersion)
		if err != nil {
			return mustMigrate, mustReindex, err
		}
	}

	// see if index need to be created or deleted and re-created
	if mustReindex || indexVersion < currentIndexDBVersion {

		mustReindex = true

		// close out current index
		poolData.dbIndex.Close()
		poolData.dbIndex = nil

		logger.Criticalf("drop index database: %s", indexDatabase)

		// erase the index completely
		err = os.RemoveAll(indexDatabase)
		if nil != err {
			return mustMigrate, mustReindex, err
		}

		// generate an empty index database
		poolData.dbIndex, _, err = getDB(indexDatabase, readOnly)
		if nil != err {
			return mustMigrate, mustReindex, err
		}

	}

	// this will be a struct type
	poolType := reflect.TypeOf(Pool)

	// get write access by using pointer + Elem()
	poolValue := reflect.ValueOf(&Pool).Elem()

	// databases
	poolData.blocksTrx = new(leveldb.Batch)
	poolData.indexTrx = new(leveldb.Batch)
	poolData.cache = newCache()
	blockDBAccess := newDA(poolData.dbBlocks, poolData.blocksTrx, poolData.cache)
	indexDBAccess := newDA(poolData.dbIndex, poolData.indexTrx, poolData.cache)
	access := []DataAccess{blockDBAccess, indexDBAccess}
	poolData.trx = newTransaction(access)

	// scan each field
	for i := 0; i < poolType.NumField(); i += 1 {

		fieldInfo := poolType.Field(i)

		prefixTag := fieldInfo.Tag.Get("prefix")
		if 1 != len(prefixTag) {
			return mustMigrate, mustReindex, fmt.Errorf("pool: %v has invalid prefix: %q", fieldInfo, prefixTag)
		}

		prefix := prefixTag[0]
		limit := []byte(nil)
		if prefix < 255 {
			limit = []byte{prefix + 1}
		}

		var dataAccess DataAccess
		switch dbName := fieldInfo.Tag.Get("database"); dbName {
		case "blocks":
			dataAccess = blockDBAccess
		case "index":
			dataAccess = indexDBAccess
		default:
			return mustMigrate, mustReindex, fmt.Errorf("pool: %v  has invalid database: %q", fieldInfo, dbName)
		}

		p := &PoolHandle{
			prefix:     prefix,
			limit:      limit,
			dataAccess: dataAccess,
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
	return mustMigrate, mustReindex, nil
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

// Finalise - close the database connection
func Finalise() {
	poolData.Lock()
	dbClose()
	poolData.Unlock()
}

// ReindexDone - called at the end of reindex
func ReindexDone() error {
	poolData.Lock()
	defer poolData.Unlock()
	return putVersion(poolData.dbIndex, currentIndexDBVersion)
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

func NewDBTransaction() (Transaction, error) {
	err := poolData.trx.Begin()
	if nil != err {
		return nil, err
	}
	return poolData.trx, nil
}
