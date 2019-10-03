// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"encoding/binary"
	"fmt"
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
	Blocks            *PoolHandle `prefix:"B"`
	BlockHeaderHash   *PoolHandle `prefix:"2"`
	BlockOwnerPayment *PoolHandle `prefix:"H"`
	BlockOwnerTxIndex *PoolHandle `prefix:"I"`
	Assets            *PoolNB     `prefix:"A"`
	Transactions      *PoolNB     `prefix:"T"`
	OwnerNextCount    *PoolHandle `prefix:"N"`
	OwnerList         *PoolHandle `prefix:"L"`
	OwnerTxIndex      *PoolHandle `prefix:"D"`
	OwnerData         *PoolHandle `prefix:"O"`
	Shares            *PoolHandle `prefix:"F"`
	ShareQuantity     *PoolHandle `prefix:"Q"`
	TestData          *PoolHandle `prefix:"Z"`
}

// Pool - the set of exported pools
var Pool pools

// for database version
var (
	versionKey    = []byte{0x00, 'V', 'E', 'R', 'S', 'I', 'O', 'N'}
	needMigration = false
)

const (
	currentBitmarksDBVersion = 0x1
)

// holds the database handle
var poolData struct {
	sync.RWMutex
	bitmarksDB    *leveldb.DB
	trx           Transaction
	bitmarksBatch *leveldb.Batch
	cache         Cache
}

var PaymentStorage struct {
	Btc P2PStorage
	Ltc P2PStorage
}

// pool access modes
const (
	ReadOnly  = true
	ReadWrite = false
)

// Initialise - open up the database connection
//
// this must be called before any pool is accessed
func Initialise(dbPrefix string, readOnly bool) error {
	poolData.Lock()
	defer poolData.Unlock()

	ok := false

	if nil != poolData.bitmarksDB {
		return fault.AlreadyInitialised
	}

	defer func() {
		if !ok {
			dbClose()
		}
	}()

	db, bitmarksDBVersion, err := initialiseBitmarksDB(dbPrefix, readOnly)
	if err != nil {
		return err
	}

	err = validateBitmarksDB(bitmarksDBVersion, readOnly)
	if err != nil {
		return err
	}

	// payment dbPrefix
	btcDatabase := dbPrefix + "-btc.leveldb"
	ltcDatabase := dbPrefix + "-ltc.leveldb"

	db, _, err = getDB(btcDatabase, readOnly)
	if nil != err {
		return err
	}
	PaymentStorage.Btc = NewLevelDBPaymentStore(db)

	db, _, err = getDB(ltcDatabase, readOnly)
	if nil != err {
		return err
	}
	PaymentStorage.Ltc = NewLevelDBPaymentStore(db)

	// this will be a struct type
	poolType := reflect.TypeOf(Pool)

	// get write access by using pointer + Elem()
	poolValue := reflect.ValueOf(&Pool).Elem()

	// databases
	poolData.bitmarksBatch = new(leveldb.Batch)
	poolData.cache = newCache()

	bitmarksDBAccess := newDA(poolData.bitmarksDB, poolData.bitmarksBatch, poolData.cache)

	access := []DataAccess{bitmarksDBAccess}
	poolData.trx = newTransaction(access)

	// scan each field
	for i := 0; i < poolType.NumField(); i += 1 {

		fieldInfo := poolType.Field(i)

		prefixTag := fieldInfo.Tag.Get("prefix")
		if 1 != len(prefixTag) {
			return fmt.Errorf("pool: %v has invalid prefix: %q", fieldInfo, prefixTag)
		}

		prefix := prefixTag[0]
		limit := []byte(nil)
		if prefix < 255 {
			limit = []byte{prefix + 1}
		}

		p := &PoolHandle{
			prefix:     prefix,
			limit:      limit,
			dataAccess: bitmarksDBAccess,
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
	return nil
}

func initialiseBitmarksDB(dbPrefix string, readOnly bool) (*leveldb.DB, int, error) {
	// bitmarksDB name
	bitmarksDatabase := dbPrefix + "-bitmarks.leveldb"

	db, bitmarksDBVersion, err := getDB(bitmarksDatabase, readOnly)
	if nil != err {
		return nil, 0, nil
	}
	poolData.bitmarksDB = db
	return db, bitmarksDBVersion, err
}

func validateBitmarksDB(bitmarksDBVersion int, readOnly bool) error {
	// ensure no database downgrade
	if bitmarksDBVersion > currentBitmarksDBVersion {
		msg := fmt.Sprintf("bitmarksDB database version: %d > current version: %d", bitmarksDBVersion, currentBitmarksDBVersion)

		logger.Critical(msg)
		return nil
	}

	// prevent readOnly from modifying the database
	if readOnly && bitmarksDBVersion != currentBitmarksDBVersion {
		msg := fmt.Sprintf("database inconsistent: bitmarksDB: %d  current: %d ", bitmarksDBVersion, currentBitmarksDBVersion)

		logger.Critical(msg)
		return nil
	}

	if 0 < bitmarksDBVersion && bitmarksDBVersion < currentBitmarksDBVersion {
		needMigration = true
	} else if 0 == bitmarksDBVersion {
		// database was empty so tag as current version
		err := putVersion(poolData.bitmarksDB, currentBitmarksDBVersion)
		if err != nil {
			return nil
		}
	}

	return nil
}

func dbClose() {
	if nil != poolData.bitmarksDB {
		if err := poolData.bitmarksDB.Close(); nil != err {
			logger.Criticalf("close bitmarkd db with error: %s", err)
		}
		poolData.bitmarksDB = nil
	}

	if nil != PaymentStorage.Btc {
		if err := PaymentStorage.Btc.Close(); nil != err {
			logger.Criticalf("close btc db with error: %s", err)
		}
		PaymentStorage.Btc = nil
	}

	if nil != PaymentStorage.Ltc {
		if err := PaymentStorage.Ltc.Close(); nil != err {
			logger.Criticalf("close btc db with error: %s", err)
		}
		PaymentStorage.Ltc = nil
	}
}

// Finalise - close the database connection
func Finalise() {
	poolData.Lock()
	dbClose()
	poolData.Unlock()
}

// return:
//   database handle
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
		e := db.Close()
		if nil != e {
			logger.Criticalf("close %s database with error: %s", name, e)
		}
		return nil, 0, err
	}

	if 4 != len(versionValue) {
		e := db.Close()
		if nil != e {
			logger.Criticalf("close %s database with error: %s", name, e)
		}
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

// IsMigrationNeed - check if bitmarks database needs migration
func IsMigrationNeed() bool {
	return needMigration
}

func NewDBTransaction() (Transaction, error) {
	err := poolData.trx.Begin()
	if nil != err {
		return nil, err
	}
	return poolData.trx, nil
}
