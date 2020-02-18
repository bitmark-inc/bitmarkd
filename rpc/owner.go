// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Owner
// -----

// Owner - type for the RPC
type Owner struct {
	log     *logger.L
	limiter *rate.Limiter
}

// Owner bitmarks
// --------------

const (
	maximumBitmarksCount = 100
)

// OwnerBitmarksArguments - arguments for RPC
type OwnerBitmarksArguments struct {
	Owner *account.Account `json:"owner"`        // base58
	Start uint64           `json:"start,string"` // first record number
	Count int              `json:"count"`        // number of records
}

// OwnerBitmarksReply - result of owner RPC
type OwnerBitmarksReply struct {
	Next uint64                    `json:"next,string"` // start value for the next call
	Data []ownership.Ownership     `json:"data"`        // list of bitmarks either issue or transfer
	Tx   map[string]BitmarksRecord `json:"tx"`          // table of tx records
}

// BitmarksRecord - can be any of the transaction records
type BitmarksRecord struct {
	Record  string      `json:"record"`
	TxId    interface{} `json:"txId,omitempty"`
	InBlock uint64      `json:"inBlock"`
	AssetId interface{} `json:"assetId,omitempty"`
	Data    interface{} `json:"data"`
}

// BlockAsset - special record for owned blocks
type BlockAsset struct {
	Number uint64 `json:"number"`
}

// Bitmarks - list bitmarks belonging to an account
func (owner *Owner) Bitmarks(arguments *OwnerBitmarksArguments, reply *OwnerBitmarksReply) error {

	if err := rateLimitN(owner.limiter, arguments.Count, maximumBitmarksCount); nil != err {
		return err
	}

	log := owner.log
	log.Infof("Owner.Bitmarks: %+v", arguments)

	ownershipData, err := ownership.ListBitmarksFor(arguments.Owner, arguments.Start, arguments.Count)
	if nil != err {
		return err
	}

	log.Debugf("ownership: %+v", ownershipData)

	// extract unique TxIds
	//   issues TxId == IssueTxId
	//   assets could be duplicates
	txIds := make(map[merkle.Digest]struct{})
	assetIds := make(map[transactionrecord.AssetIdentifier]struct{})
	current := uint64(0)
	for _, r := range ownershipData {
		txIds[r.TxId] = struct{}{}
		txIds[r.IssueTxId] = struct{}{}
		switch r.Item {
		case ownership.OwnedAsset:
			ai := r.AssetId
			if nil == ai {
				log.Criticalf("asset id is nil: %+v", r)
				logger.Panicf("asset id is nil: %+v", r)
			}
			assetIds[*r.AssetId] = struct{}{}
		case ownership.OwnedBlock:
			if nil == r.BlockNumber {
				log.Criticalf("block number is nil: %+v", r)
				logger.Panicf("blockNumber is nil: %+v", r)
			}
		case ownership.OwnedShare:
			ai := r.AssetId
			if nil == ai {
				log.Criticalf("asset id is nil: %+v", r)
				logger.Panicf("asset id is nil: %+v", r)
			}
			assetIds[*r.AssetId] = struct{}{}
		default:
			log.Criticalf("unsupported item type: %d", r.Item)
			logger.Panicf("unsupported item type: %d", r.Item)
		}
		current = r.N
	}

	records := make(map[string]BitmarksRecord)

	for txId := range txIds {

		log.Debugf("txId: %v", txId)

		inBlock, transaction := storage.Pool.Transactions.GetNB(txId[:])
		if nil == transaction {
			return fault.LinkToInvalidOrUnconfirmedTransaction
		}

		tx, _, err := transactionrecord.Packed(transaction).Unpack(mode.IsTesting())
		if nil != err {
			return err
		}

		record, ok := transactionrecord.RecordName(tx)
		if !ok {
			log.Errorf("problem tx: %+v", tx)
			return fault.LinkToInvalidOrUnconfirmedTransaction
		}
		textTxId, err := txId.MarshalText()
		if nil != err {
			return err
		}

		records[string(textTxId)] = BitmarksRecord{
			Record:  record,
			TxId:    txId,
			InBlock: inBlock,
			Data:    tx,
		}
	}

asset_loop:
	for assetId := range assetIds {

		log.Debugf("asset id: %v", assetId)

		var nnn transactionrecord.AssetIdentifier
		if nnn == assetId {
			records["00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"] = BitmarksRecord{
				Record: "Block",
				//AssetId: assetId,
				Data: BlockAsset{
					Number: 0,
				},
			}
			continue asset_loop
		}

		inBlock, transaction := storage.Pool.Assets.GetNB(assetId[:])
		if nil == transaction {
			return fault.AssetNotFound
		}

		tx, _, err := transactionrecord.Packed(transaction).Unpack(mode.IsTesting())
		if nil != err {
			return err
		}

		record, ok := transactionrecord.RecordName(tx)
		if !ok {
			return fault.AssetNotFound
		}
		textAssetId, err := assetId.MarshalText()
		if nil != err {
			return err
		}

		records[string(textAssetId)] = BitmarksRecord{
			Record:  record,
			InBlock: inBlock,
			AssetId: assetId,
			Data:    tx,
		}
	}

	reply.Data = ownershipData
	reply.Tx = records

	// if no record were found the just return Next as zero
	// otherwise the next possible number
	if 0 == current {
		reply.Next = 0
	} else {
		reply.Next = current + 1
	}
	return nil
}
