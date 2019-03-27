// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package block

import (
	"encoding/binary"
	"reflect"
	"sync"

	"github.com/bitmark-inc/bitmarkd/background"
	"github.com/bitmark-inc/bitmarkd/blockdigest"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

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

// isTxWipedOut will check whether a transaction is cleaned up from the index db
func isTxWipedOut(txId merkle.Digest) bool {
	if storage.Pool.OwnerData.Has(txId[:]) {
		globalData.log.Error("tx data is not cleaned")
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
			globalData.log.Error("owner tx index is not cleaned")
			return false
		}
	case transactionrecord.BitmarkTransfer:
		txIndexKey := append(tx.GetOwner().Bytes(), txId[:]...)
		if storage.Pool.OwnerTxIndex.Has(txIndexKey) {
			globalData.log.Error("owner tx index is not cleaned")
			return false
		}
	default:
		globalData.log.Critical("invalid type of transaction")
		return false
	}

	return true
}

func validateTransactionData(header *blockrecord.Header, digest blockdigest.Digest, data []byte) error {
	oldBlockOwnerTxs := map[string]struct{}{}
	priorTxOwnerTxs := map[string]struct{}{}
	for i := header.TransactionCount; i > 0; i-- {
		transaction, n, err := transactionrecord.Packed(data).Unpack(mode.IsTesting())
		if err != nil {
			globalData.log.Error("can not unpack transaction")
			return err
		}

		txId := merkle.NewDigest(data[:n])

		switch tx := transaction.(type) {
		case *transactionrecord.BlockFoundation:
			foundationTxId := blockrecord.FoundationTxId(header, digest)
			globalData.log.Debugf("validate whether the foundation transaction indexed. foundationTxId: %s", foundationTxId)
			if !storage.Pool.Transactions.Has(foundationTxId[:]) {
				globalData.log.Error("foundation tx is not indexed")
				return fault.ErrTransactionIsNotIndexed
			}

			if _, ok := oldBlockOwnerTxs[txId.String()]; !ok {
				globalData.log.Debugf("validate whether the ownership indexed. foundationTxId: %s", foundationTxId)
				if !storage.Pool.BlockOwnerTxIndex.Has(foundationTxId[:]) {
					globalData.log.Error("ownership is not indexed")
					return fault.ErrOwnershipIsNotIndexed
				}

				packedPayment, err := tx.Payments.Pack(mode.IsTesting())
				if err != nil {
					globalData.log.Error("fail to get packed payments")
					return err
				}

				blockNumberKey := make([]byte, 8)
				binary.BigEndian.PutUint64(blockNumberKey, header.Number)
				globalData.log.Debugf("validate whether the payment info identical. foundationTxId: %s", foundationTxId)
				if !reflect.DeepEqual(storage.Pool.BlockOwnerPayment.Get(blockNumberKey[:]), packedPayment) {
					globalData.log.Error("payment info data is not consistent")
					return fault.ErrDataInconsistent
				}
			}

		case *transactionrecord.BlockOwnerTransfer:
			globalData.log.Debugf("validate whether the owner transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.ErrTransactionIsNotIndexed
			}

			globalData.log.Debugf("validate whether the previous ownership cleaned. txId: %s", txId)
			if storage.Pool.BlockOwnerTxIndex.Has(tx.Link[:]) {
				globalData.log.Error("ownership is not cleaned")
				return fault.ErrOwnershipIsNotCleaned
			}
			oldBlockOwnerTxs[tx.Link.String()] = struct{}{}

			if _, ok := oldBlockOwnerTxs[txId.String()]; !ok {
				// validate ownership indexed only if the txId is not added into olderBlockOwnerTxs
				globalData.log.Debugf("validate whether the ownership indexed. txId: %s", txId)
				blockNumberKey := storage.Pool.BlockOwnerTxIndex.Get(txId[:])
				if blockNumberKey == nil {
					globalData.log.Error("ownership is not indexed")
					return fault.ErrOwnershipIsNotIndexed
				}

				packedPayment, err := tx.Payments.Pack(mode.IsTesting())
				if err != nil {
					globalData.log.Error("fail to get packed payments")
					return err
				}

				binary.BigEndian.PutUint64(blockNumberKey, header.Number)
				globalData.log.Debugf("validate whether the payment info identical. txId: %s", txId)
				if !reflect.DeepEqual(storage.Pool.BlockOwnerPayment.Get(blockNumberKey[:]), packedPayment) {
					globalData.log.Error("payment info data is not consistent")
					return fault.ErrDataInconsistent
				}
			}

		case *transactionrecord.AssetData:
			globalData.log.Debugf("validate whether the asset indexed. txId: %s", txId)
			assetId := tx.AssetId()
			if !storage.Pool.Assets.Has(assetId[:]) {
				globalData.log.Error("asset is not indexed")
				return fault.ErrAssetIsNotIndexed
			}
		case *transactionrecord.BitmarkIssue:
			globalData.log.Debugf("validate whether the issue transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.ErrTransactionIsNotIndexed
			}

		case transactionrecord.BitmarkTransfer:
			globalData.log.Debugf("validate whether the transfer transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.ErrTransactionIsNotIndexed
			}

			globalData.log.Debugf("validate whether the prior transaction wiped out. txId: %s", txId)
			if !isTxWipedOut(tx.GetLink()) {
				return fault.ErrTransactionIsNotCleaned
			}

			if _, ok := priorTxOwnerTxs[txId.String()]; !ok {
				// if a tx is not in the list of prior transaction, it should be the latest tx.
				txIndexKey := append(tx.GetOwner().Bytes(), txId[:]...)
				count := storage.Pool.OwnerTxIndex.Get(txIndexKey)

				ownerListKey := append(tx.GetOwner().Bytes(), count[:]...)
				txIdFromList := storage.Pool.OwnerList.Get(ownerListKey)

				if !reflect.DeepEqual(txIdFromList[:], txId[:]) {
					globalData.log.Error("tx record inconsistent")
					return fault.ErrDataInconsistent
				}
				priorTxOwnerTxs[txId.String()] = struct{}{}
			}

		case *transactionrecord.BitmarkShare, *transactionrecord.ShareGrant, *transactionrecord.ShareSwap:
			globalData.log.Debugf("validate whether the share transaction indexed. txId: %s", txId)
			if !storage.Pool.Transactions.Has(txId[:]) {
				globalData.log.Error("tx is not indexed")
				return fault.ErrTransactionIsNotIndexed
			}
		default:
			globalData.log.Error("unexpected transaction records")
			return fault.ErrUnexpectedTransaction
		}
		data = data[n:]
	}

	return nil
}

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

		h, d, data, err := blockrecord.ExtractHeader(blockData)
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

// setup the current block data
func Initialise(recover bool) error {
	globalData.Lock()
	defer globalData.Unlock()

	// no need to start if already started
	if globalData.initialised {
		return fault.ErrAlreadyInitialised
	}

	log := logger.New("block")
	globalData.log = log
	log.Info("starting…")

	// check storage is initialised
	if nil == storage.Pool.Blocks {
		log.Critical("storage pool is not initialised")
		return fault.ErrNotInitialised
	}

	if recover {
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

// shutdown the block system
func Finalise() error {

	if !globalData.initialised {
		return fault.ErrNotInitialised
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
