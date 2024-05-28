// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"sync"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
	"github.com/syndtr/goleveldb/leveldb"
	ldb_opt "github.com/syndtr/goleveldb/leveldb/opt"
)

// exported storage pools
//
// note all must be exported (i.e. initial capital) or initialisation will panic
type pools struct {
	Blocks            Handle `prefix:"B" pool:"PoolHandle"`
	BlockHeaderHash   Handle `prefix:"2" pool:"PoolHandle"`
	BlockOwnerPayment Handle `prefix:"H" pool:"PoolHandle"`
	BlockOwnerTxIndex Handle `prefix:"I" pool:"PoolHandle"`
	Assets            Handle `prefix:"A" pool:"PoolNB"`
	Transactions      Handle `prefix:"T" pool:"PoolNB"`
	OwnerNextCount    Handle `prefix:"N" pool:"PoolHandle"`
	OwnerList         Handle `prefix:"L" pool:"PoolHandle"`
	OwnerTxIndex      Handle `prefix:"D" pool:"PoolHandle"`
	OwnerData         Handle `prefix:"O" pool:"PoolHandle"`
	Shares            Handle `prefix:"F" pool:"PoolHandle"`
	ShareQuantity     Handle `prefix:"Q" pool:"PoolHandle"`
	TestData          Handle `prefix:"Z" pool:"PoolHandle"`
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
	bitmarksDBName           = "bitmarks"
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

	if poolData.bitmarksDB != nil {
		return fault.AlreadyInitialised
	}

	defer func() {
		if !ok {
			dbClose()
		}
	}()

	bitmarksDBVersion, err := openBitmarkdDB(dbPrefix, readOnly)
	if err != nil {
		return err
	}

	err = validateBitmarksDBVersion(bitmarksDBVersion, readOnly)
	if err != nil {
		return err
	}

	err = setupBitmarksDB()
	if err != nil {
		return err
	}

	// payment dbPrefix
	btcDatabase := dbPrefix + "-btc.leveldb"
	ltcDatabase := dbPrefix + "-ltc.leveldb"

	db, _, err := getDB(btcDatabase, readOnly)
	if err != nil {
		return err
	}
	PaymentStorage.Btc = NewLevelDBPaymentStore(db)

	db, _, err = getDB(ltcDatabase, readOnly)
	if err != nil {
		return err
	}
	PaymentStorage.Ltc = NewLevelDBPaymentStore(db)

	ok = true // prevent db close
	return nil
}

func setupBitmarksDB() error {
	bitmarksDBAccess := setupBitmarksDBTransaction()

	err := setupPools(bitmarksDBAccess)
	if err != nil {
		return err
	}

	return nil
}

func setupBitmarksDBTransaction() Access {
	poolData.bitmarksBatch = new(leveldb.Batch)
	poolData.cache = newCache()
	bitmarksDBAccess := newDA(poolData.bitmarksDB, poolData.bitmarksBatch, poolData.cache)
	poolData.trx = newTransaction([]Access{bitmarksDBAccess})

	return bitmarksDBAccess
}

func setupPools(bitmarksDBAccess Access) error {
	// this will be a struct type
	poolType := reflect.TypeOf(Pool)
	// get write access by using pointer + Elem()
	poolValue := reflect.ValueOf(&Pool).Elem()

	// scan each field
	for i := 0; i < poolType.NumField(); i += 1 {
		fieldInfo := poolType.Field(i)
		prefixTag := fieldInfo.Tag.Get("prefix")
		poolTag := fieldInfo.Tag.Get("pool")

		if len(prefixTag) != 1 || poolTag == "" {
			return fmt.Errorf("pool: %v has invalid prefix: %q, poolTag: %s", fieldInfo, prefixTag, poolTag)
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

		if poolTag == "PoolNB" {
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
	return nil
}

func openBitmarkdDB(dbPrefix string, readOnly bool) (int, error) {
	name := fmt.Sprintf("%s-%s.leveldb", dbPrefix, bitmarksDBName)

	db, version, err := getDB(name, readOnly)
	if err != nil {
		return 0, err
	}
	poolData.bitmarksDB = db

	return version, err
}

func validateBitmarksDBVersion(bitmarksDBVersion int, readOnly bool) error {
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
	} else if bitmarksDBVersion == 0 {
		// database was empty so tag as current version
		err := putVersion(poolData.bitmarksDB, currentBitmarksDBVersion)
		if err != nil {
			return nil
		}
	}

	return nil
}

func dbClose() {
	if poolData.bitmarksDB != nil {
		if err := poolData.bitmarksDB.Close(); err != nil {
			logger.Criticalf("close bitmarkd db with error: %s", err)
		}
		poolData.bitmarksDB = nil
	}

	if PaymentStorage.Btc != nil {
		if err := PaymentStorage.Btc.Close(); err != nil {
			logger.Criticalf("close btc db with error: %s", err)
		}
		PaymentStorage.Btc = nil
	}

	if PaymentStorage.Ltc != nil {
		if err := PaymentStorage.Ltc.Close(); err != nil {
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
//
//	database handle
//	version number
func getDB(name string, readOnly bool) (*leveldb.DB, int, error) {
	opt := &ldb_opt.Options{
		ErrorIfExist:   false,
		ErrorIfMissing: readOnly,
		ReadOnly:       readOnly,
	}

	db, err := leveldb.OpenFile(name, opt)
	if err != nil {
		return nil, 0, err
	}

	versionValue, err := db.Get(versionKey, nil)
	if leveldb.ErrNotFound == err {
		return db, 0, nil
	} else if err != nil {
		e := db.Close()
		if e != nil {
			logger.Criticalf("close %s database with error: %s", name, e)
		}
		return nil, 0, err
	}

	if len(versionValue) != 4 {
		e := db.Close()
		if e != nil {
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
	if err != nil {
		return nil, err
	}
	return poolData.trx, nil
}
