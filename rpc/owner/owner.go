// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package owner

import (
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/ratelimit"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Owner
// -----

// Owner - type for the RPC
type Owner struct {
	Log              *logger.L
	Limiter          *rate.Limiter
	PoolTransactions storage.Handle
	PoolAssets       storage.Handle
	Ownership        ownership.Ownership
}

// Owner bitmarks
// --------------

const (
	MaximumBitmarksCount = 100
	rateLimitOwner       = 200
	rateBurstOwner       = 100
)

// BitmarksArguments - arguments for RPC
type BitmarksArguments struct {
	Owner *account.Account `json:"owner"`        // base58
	Start uint64           `json:"Start,string"` // first record number
	Count int              `json:"count"`        // number of records
}

// BitmarksReply - result of owner RPC
type BitmarksReply struct {
	Next uint64                    `json:"next,string"` // Start value for the next call
	Data []ownership.Record        `json:"data"`        // list of bitmarks either issue or transfer
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

func New(log *logger.L, pools reservoir.Handles, os ownership.Ownership) *Owner {
	return &Owner{
		Log:              log,
		Limiter:          rate.NewLimiter(rateLimitOwner, rateBurstOwner),
		PoolTransactions: pools.Transactions,
		PoolAssets:       pools.Assets,
		Ownership:        os,
	}
}

// Bitmarks - list bitmarks belonging to an account
func (owner *Owner) Bitmarks(arguments *BitmarksArguments, reply *BitmarksReply) error {

	if err := ratelimit.LimitN(owner.Limiter, arguments.Count, MaximumBitmarksCount); nil != err {
		return err
	}

	log := owner.Log
	log.Infof("Owner.Bitmarks: %+v", arguments)

	ownershipData, err := owner.Ownership.ListBitmarksFor(arguments.Owner, arguments.Start, arguments.Count)
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

		inBlock, transaction := owner.PoolTransactions.GetNB(txId[:])
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

assetsLoop:
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
			continue assetsLoop
		}

		inBlock, transaction := owner.PoolAssets.GetNB(assetId[:])
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
