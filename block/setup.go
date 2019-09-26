// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"reflect"
	"sync"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// BlockValidationCounts - number of blocks to validate on startup
const BlockValidationCounts = 10

// globals for background process
type blockData struct {
	sync.RWMutex // to allow locking

	log *logger.L

	rebuild bool       // set if all indexes are being rebuild
	blk     blockstore // for sequencing block storage

	// for background
	background *background.T

	// set once during initialise
	initialised bool
}

// global data
var globalData blockData

var (
	priorBlockOwnerTxs = map[string]struct{}{}
	priorTxOwnerTxs    = map[string]struct{}{}
)

// validateTxOwnerRecords will check
// (1) whether the `OwnerTxIndex` has a key of [owner+txId]
// (2) whether the `OwnerList` has a key of [owner+{value of (1)}]
// (3) whether the value of (2) is the txId
func validateTxOwnerRecords(txId merkle.Digest, owner *account.Account) error {
	txIndexKey := append(owner.Bytes(), txId[:]...)
	count := storage.Pool.OwnerTxIndex.Get(txIndexKey)

	ownerListKey := append(owner.Bytes(), count[:]...)
	txIdFromList := storage.Pool.OwnerList.Get(ownerListKey)

	if !reflect.DeepEqual(txIdFromList[:], txId[:]) {
		return fault.DataInconsistent
	}
	return nil
}

// validateBlockOwnerRecord will check
// (1) whether `BlockOwnerTxIndex` has a key of txId
// (2) whether the value of `BlockOwnerPayment` is identical to the packed payments
func validateBlockOwnerRecord(txId merkle.Digest, payments currency.Map) error {
	blockNumberKey := storage.Pool.BlockOwnerTxIndex.Get(txId[:])

	if len(blockNumberKey) == 0 {
		globalData.log.Error("ownership is not indexed")
		return fault.OwnershipIsNotIndexed
	}

	packedPayment, err := payments.Pack(mode.IsTesting())
	if err != nil {
		globalData.log.Error("fail to get packed payments")
		return err
	}

	globalData.log.Debugf("validate whether the payment info identical. txId: %s", txId)
	if !reflect.DeepEqual(storage.Pool.BlockOwnerPayment.Get(blockNumberKey[:]), packedPayment) {
		globalData.log.Error("payment info data is not consistent")
		return fault.DataInconsistent
	}
	return nil
}

// isTxWipedOut will check whether a transaction is deleted up from the index db
func isTxWipedOut(txId merkle.Digest) bool {
	if storage.Pool.OwnerData.Has(txId[:]) {
		globalData.log.Error("tx data is not deleted")
		return false
	}

	_, packed := storage.Pool.Transactions.GetNB(txId[:])
	transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
	if err != nil {
		globalData.log.Errorf("can not fetch transaction. error: %s", err)
		return false
	}

	switch tx := transaction.(type) {
	case *transactionrecord.BitmarkIssue:
		txIndexKey := append(tx.Owner.Bytes(), txId[:]...)
		if storage.Pool.OwnerTxIndex.Has(txIndexKey) {
			globalData.log.Error("owner tx index is not deleted")
			return false
		}
	case transactionrecord.BitmarkTransfer:
		txIndexKey := append(tx.GetOwner().Bytes(), txId[:]...)
		if storage.Pool.OwnerTxIndex.Has(txIndexKey) {
			globalData.log.Error("owner tx index is not deleted")
			return false
		}
	default:
		globalData.log.Critical("invalid type of transaction")
		return false
	}

	return true
}

// validateTransactionData will validate the block records by go through the transaction records
func validateTransactionData(header *blockrecord.Header, digest blockdigest.Digest, data []byte) error {
	for i := header.TransactionCount; i > 0; i-- {
		transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
		if err != nil {
			globalData.log.Error("can not unpack transaction")
			return err
		}

		txId := merkle.NewDigest(data[:n])

		switch tx := transaction.(type) {
		case *transactionrecord.OldBaseData:
			globalData.log.Warnf("not processing base record: %+v", tx)
		case *transactionrecord.BlockFoundation:
			// use foundationTxId instead of txId for block foundation check
			foundationTxId := blockrecord.FoundationTxId(header.Number, digest)

			globalData.log.Debugf("validate whether the foundation transaction indexed. foundationTxId: %s", foundationTxId)
			if !storage.Pool.Transactions.Has(foundationTxId[:]) {
				globalData.log.Error("foundation tx is not indexed")
				return fault.TransactionIsNotIndexed
			}

			if _, ok := priorBlockOwnerTxs[foundationTxId.String()]; !ok {
				// validate ownership indexed only if the txId is not in priorBlockOwnerTxs
				globalData.log.Debugf("validate whether the ownership indexed. foundationTxId: %s", foundationTxId)
				if err := validateBlockOwnerRecord(foundationTxId, tx.Payments); err != nil {
					globalData.log.Error("block ownership validation failed")
					return err
				}
			}

		case *transactionrecord.BlockOwnerTransfer:
			globalData.log.Debugf("validate whether the owner transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.TransactionIsNotIndexed
			}

			globalData.log.Debugf("validate whether the previous ownership deleted. txId: %s", txId)
			if storage.Pool.BlockOwnerTxIndex.Has(tx.Link[:]) {
				globalData.log.Error("ownership is not deleted")
				return fault.PreviousOwnershipWasNotDeleted
			}

			if _, ok := priorBlockOwnerTxs[txId.String()]; !ok {
				// validate ownership indexed only if the txId is not in priorBlockOwnerTxs
				globalData.log.Debugf("validate whether the block ownership indexed. txId: %s", txId)
				if err := validateBlockOwnerRecord(txId, tx.Payments); err != nil {
					globalData.log.Error("block ownership validation failed")
					return err
				}
			}
			// add the prior block ownership tx id into map
			priorBlockOwnerTxs[tx.Link.String()] = struct{}{}

		case *transactionrecord.AssetData:
			globalData.log.Debugf("validate whether the asset indexed. txId: %s", txId)
			assetId := tx.AssetId()
			if !storage.Pool.Assets.Has(assetId[:]) {
				globalData.log.Error("asset is not indexed")
				return fault.AssetIsNotIndexed
			}
		case *transactionrecord.BitmarkIssue:
			globalData.log.Debugf("validate whether the issue transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.TransactionIsNotIndexed
			}

			if _, ok := priorTxOwnerTxs[txId.String()]; !ok {
				// validate tx ownership indexed only if the txId is not in priorBlockOwnerTxs
				if err := validateTxOwnerRecords(txId, tx.Owner); err != nil {
					globalData.log.Error("transaction ownership validation failed")
					return err
				}
			}

		case transactionrecord.BitmarkTransfer:
			globalData.log.Debugf("validate whether the transfer transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.TransactionIsNotIndexed
			}

			globalData.log.Debugf("validate whether the prior transaction wiped out. txId: %s", txId)
			if !isTxWipedOut(tx.GetLink()) {
				return fault.PreviousTransactionWasNotDeleted
			}

			if _, ok := priorTxOwnerTxs[txId.String()]; !ok && nil != tx.GetOwner() {
				// validate tx ownership indexed only if the txId is not in priorBlockOwnerTxs
				if err := validateTxOwnerRecords(txId, tx.GetOwner()); err != nil {
					globalData.log.Error("transaction ownership validation failed")
					return err
				}
			}
			// add the prior tx id into map
			priorTxOwnerTxs[tx.GetLink().String()] = struct{}{}

		//nolint:ignore SA4020 XXX: unreachable case clause here
		case *transactionrecord.BitmarkShare, *transactionrecord.ShareGrant, *transactionrecord.ShareSwap:
			globalData.log.Debugf("validate whether the share transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.TransactionIsNotIndexed
			}
		default:
			globalData.log.Errorf("unexpected transaction record: %+v", tx)
			return fault.UnexpectedTransactionRecord
		}
		data = data[n:]
	}

	return nil
}

// validateAndReturnLastBlock will validate index db according to the block db.
// If all the validation are passed, it returns the latest block.
func validateAndReturnLastBlock(last storage.Element) (*blockrecord.Header, blockdigest.Digest, error) {
	var header *blockrecord.Header
	var digest blockdigest.Digest

	blocks := [][]byte{last.Value}
	lastBlockNumber := binary.BigEndian.Uint64(last.Key)
	lastCheckedBlockNumber := lastBlockNumber - BlockValidationCounts
	if lastCheckedBlockNumber < 1 {
		lastCheckedBlockNumber = 1
	}

	for blockNumber := lastBlockNumber - 1; blockNumber > lastCheckedBlockNumber; blockNumber-- {
		blockNumberKey := make([]byte, 8)
		binary.BigEndian.PutUint64(blockNumberKey, blockNumber)
		block := storage.Pool.Blocks.Get(blockNumberKey)
		blocks = append(blocks, block) // append
	}

	for i, blockData := range blocks {
		var data []byte
		var err error

		h, d, data, err := blockrecord.ExtractHeader(blockData, 0, false)
		if err != nil {
			globalData.log.Error("can not extract header")
			return h, d, err
		}

		// set the last block header and digest by index 0, since we validate blocks by descending order
		if i == 0 {
			header = h
			digest = d
		}

		globalData.log.Infof("validate block. block number: %d, transaction count: %d", h.Number, h.TransactionCount)

		if err := validateTransactionData(h, d, data); err != nil {
			return h, d, err
		}
	}

	return header, digest, nil
}

// Initialise - setup the current block data
func Initialise(migrate, reindex bool) error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.AlreadyInitialised
	}

	log := logger.New("block")
	globalData.log = log
	log.Info("starting…")

	// check storage is initialised
	if nil == storage.Pool.Blocks {
		log.Critical("storage pool is not initialised")
		return fault.NotInitialised
	}

	if migrate {
		log.Info("start block migration…")
		globalData.rebuild = true
		globalData.Unlock()
		err := doBlockHeaderHash()
		globalData.Lock()
		if nil != err {
			log.Criticalf("blocks migration error: %s", err)
			return err
		}
		log.Info("block migration completed")
	}

	if reindex {
		log.Warn("start index rebuild…")
		globalData.rebuild = true
		globalData.Unlock()
		err := doRecovery()
		globalData.Lock()
		if nil != err {
			log.Criticalf("index rebuild error: %s", err)
			return err
		}
		log.Warn("index rebuild completed")
	}

	// ensure not in rebuild mode
	globalData.rebuild = false

	// detect if any blocks on file
	if last, ok := storage.Pool.Blocks.LastElement(); ok {

		// start validating block indexes
		header, digest, err := validateAndReturnLastBlock(last)
		if nil != err {
			log.Criticalf("failed to validate blocks from storage  error: %s", err)
			return err
		}

		height := header.Number
		blockheader.Set(height, digest, header.Version, header.Timestamp)

		log.Infof("highest block from storage: %d", height)
	}

	// initialise background tasks
	if err := globalData.blk.initialise(); nil != err {
		return err
	}

	// all data initialised
	globalData.initialised = true

	// start background processes
	log.Info("start background…")

	processes := background.Processes{
		&globalData.blk,
	}

	globalData.background = background.Start(processes, log)

	return nil
}

// Finalise - shutdown the block system
func Finalise() error {

	if !globalData.initialised {
		return fault.NotInitialised
	}

	globalData.log.Info("shutting down…")
	globalData.log.Flush()

	globalData.background.Stop()

	// finally...
	globalData.initialised = false

	globalData.log.Info("finished")
	globalData.log.Flush()

	return nil
}

// LastBlockHash - return the last block hash in hex string if found
func LastBlockHash() string {

	log := globalData.log

	if last, ok := storage.Pool.Blocks.LastElement(); ok {

		_, digest, _, err := blockrecord.ExtractHeader(last.Value, 0, false)
		if nil != err {
			log.Criticalf("failed to unpack block: %d from storage error: %s", binary.BigEndian.Uint64(last.Key), err)
			return ""
		}

		return digest.String()
	}

	return ""
}
