// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package storage

import (
	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/syndtr/goleveldb/leveldb"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/logger"
)

var CheckpointKey = []byte("checkpoint")

type P2PStorage interface {
	DB() *leveldb.DB
	GetHash(height int32) (*chainhash.Hash, error)
	GetHeight(hash *chainhash.Hash) (int32, error)
	StoreBlock(height int32, hash *chainhash.Hash) error
	GetCheckpoint() (*chainhash.Hash, error)
	SetCheckpoint(height int32) error
	RollbackTo(deleteFrom, deleteTo int32) error
	Close() error
}

type PaymentTable struct {
	prefix   byte
	limit    []byte
	database *leveldb.DB
}

// prepend the prefix onto the key
func (p *PaymentTable) prefixKey(key []byte) []byte {
	prefixedKey := make([]byte, 1, len(key)+1)
	prefixedKey[0] = p.prefix
	return append(prefixedKey, key...)
}

// store a key/value bytes pair to the database
func (p *PaymentTable) Put(key []byte, value []byte) error {
	if nil == p.database {
		return fault.DatabaseIsNotSet
	}
	return p.database.Put(p.prefixKey(key), value, nil)
}

// remove a key from the database
func (p *PaymentTable) Delete(key []byte) error {
	if nil == p.database {
		return fault.DatabaseIsNotSet
	}
	return p.database.Delete(p.prefixKey(key), nil)
}

// read a value for a given key
//
// this returns the actual element - copy the result if it must be preserved
func (p *PaymentTable) Get(key []byte) []byte {
	if nil == p.database {
		return nil
	}
	value, err := p.database.Get(p.prefixKey(key), nil)
	if nil != err {
		return nil
	}
	return value
}

// Check if a key exists
func (p *PaymentTable) Has(key []byte) bool {
	if nil == p.database {
		return false
	}
	value, _ := p.database.Has(p.prefixKey(key), nil)
	return value
}

type LevelDBPaymentStore struct {
	db  *leveldb.DB
	log *logger.L

	tableMap map[string]string
}

func NewLevelDBPaymentStore(db *leveldb.DB) *LevelDBPaymentStore {
	log := logger.New("storage")

	return &LevelDBPaymentStore{
		db:  db,
		log: log,
		tableMap: map[string]string{
			"hash":    "b",
			"height":  "h",
			"payment": "p",
			"receipt": "r",
		},
	}
}

func (l *LevelDBPaymentStore) DB() *leveldb.DB {
	return l.db
}

func (l *LevelDBPaymentStore) Table(tableName string) *PaymentTable {
	b := l.tableMap[tableName][0]
	var limit []byte
	if b < 255 {
		limit = []byte{b + 1}
	}

	return &PaymentTable{
		prefix:   b,
		limit:    limit,
		database: l.db,
	}
}

// GetHeight returns height for a specific hash
func (l *LevelDBPaymentStore) GetHeight(hash *chainhash.Hash) (int32, error) {
	if hash == nil {
		return -1, fault.HashCannotBeNil
	}

	heightByte := l.Table("hash").Get(hash.CloneBytes())
	if heightByte == nil {
		return -1, nil
	}

	h, err := strconv.ParseInt(string(heightByte), 16, 32)
	if err != nil {
		return -1, err
	}

	return int32(h), nil
}

// GetCheckpoint returns the hash of the last saved checkpoint in the storage
func (l *LevelDBPaymentStore) GetCheckpoint() (*chainhash.Hash, error) {
	h := l.Table("hash").Get(CheckpointKey)
	return chainhash.NewHash(h)
}

// SetCheckpoint saves the hash of a give height as a checkpoint
func (l *LevelDBPaymentStore) SetCheckpoint(height int32) error {
	hash := l.Table("height").Get([]byte(fmt.Sprintf("%08x", height)))
	if hash == nil {
		return fault.BlockHeightNotFound
	}

	return l.Table("hash").Put(CheckpointKey, hash)
}

// GetHash returns the hash of a give height
func (l *LevelDBPaymentStore) GetHash(height int32) (*chainhash.Hash, error) {
	b := l.Table("height").Get([]byte(fmt.Sprintf("%08x", height)))
	if b == nil {
		return nil, fault.HashNotFound
	}
	return chainhash.NewHash(b)
}

// StoreBlock saves a pair of hash and its height
func (l *LevelDBPaymentStore) StoreBlock(height int32, hash *chainhash.Hash) error {
	if err := l.Table("hash").Put(hash.CloneBytes(), []byte(fmt.Sprintf("%08x", height))); err != nil {
		return err
	}
	if err := l.Table("height").Put([]byte(fmt.Sprintf("%08x", height)), hash.CloneBytes()); err != nil {
		return err
	}
	return nil
}

// RollbackTo deletes blocks from a block down a another
func (l *LevelDBPaymentStore) RollbackTo(deleteFrom, deleteTo int32) error {
	if deleteFrom <= deleteTo {
		return fault.IncorrectBlockRangeToRollback
	}
	for i := deleteFrom; i > deleteTo; i-- {
		l.log.Debugf("Delete block: %d", i)
		heightsByte := []byte(fmt.Sprintf("%08x", i))
		hashByte := l.Table("height").Get(heightsByte)

		if err := l.Table("height").Delete(heightsByte); err != nil {
			return err
		}
		if err := l.Table("hash").Delete(hashByte); err != nil {
			return err
		}
	}
	return nil
}

func (l *LevelDBPaymentStore) Close() error {
	return l.db.Close()
}
