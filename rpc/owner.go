// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/binary"
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
// -------

type Owner struct {
	log *logger.L
}

// Owner bitmarks
// --------------

type OwnerBitmarksArguments struct {
	Owner *account.Account `json:"owner"`        // base58
	Start uint64           `json:"start,string"` // first record number
	Count int              `json:"count"`        // number of records
}

type OwnerBitmarksReply struct {
	Next uint64                    `json:"next,string"` // start value for the next call
	Data []ownership.Ownership     `json:"data"`        // list of bitmarks either issue or transfer
	Tx   map[string]BitmarksRecord `json:"tx"`          // table of tx records
}

// can be any of the transaction records
type BitmarksRecord struct {
	Record     string      `json:"record"`
	TxId       interface{} `json:"txId,omitempty"`
	InBlock    uint64      `json:inBlock"`
	AssetIndex interface{} `json:"index,omitempty"`
	Data       interface{} `json:"data"`
}

type BlockAsset struct {
	Number uint64 `json:"number"`
}

func (owner *Owner) Bitmarks(arguments *OwnerBitmarksArguments, reply *OwnerBitmarksReply) error {
	log := owner.log
	log.Infof("Owner.Bitmarks: %+v", arguments)

	if arguments.Count <= 0 || arguments.Count > 100 {
		return fault.ErrInvalidCount
	}

	ownershipData, err := ownership.ListBitmarksFor(arguments.Owner, arguments.Start, arguments.Count)
	if nil != err {
		return err
	}

	log.Infof("ownership: %+v", ownershipData)

	// extract unique TxIds
	//   issues TxId == IssueTxId
	//   assets could be duplicates
	txIds := make(map[merkle.Digest]struct{})
	assetIndexes := make(map[transactionrecord.AssetIndex]struct{})
	current := uint64(0)
	for _, r := range ownershipData {
		txIds[r.TxId] = struct{}{}
		txIds[r.IssueTxId] = struct{}{}
		switch r.Item {
		case ownership.OwnedAsset:
			assetIndexes[r.AssetIndex] = struct{}{}
		case ownership.OwnedBlock:
		default:
			logger.Panicf("unsupported item type: %d", r.Item)
		}
		current = r.N
	}

	records := make(map[string]BitmarksRecord)

	for txId := range txIds {

		log.Infof("txId: %v", txId)

		inBlockBuffer, transaction := storage.Pool.Transactions.GetSplit2(txId[:], 8)
		if nil == transaction {
			return fault.ErrLinkToInvalidOrUnconfirmedTransaction
		}

		inBlock := binary.BigEndian.Uint64(inBlockBuffer)

		tx, _, err := transactionrecord.Packed(transaction).Unpack(mode.IsTesting())
		if nil != err {
			return err
		}

		record, ok := transactionrecord.RecordName(tx)
		if !ok {
			return fault.ErrLinkToInvalidOrUnconfirmedTransaction
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
	for assetIndex := range assetIndexes {

		log.Infof("assetIndex: %v", assetIndex)

		var nnn transactionrecord.AssetIndex
		if nnn == assetIndex {
			records["00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"] = BitmarksRecord{
				Record: "Block",
				//AssetIndex: assetIndex,
				Data: BlockAsset{
					Number: 0,
				},
			}
			continue asset_loop
		}

		transaction := storage.Pool.Assets.Get(assetIndex[:])
		if nil == transaction {
			return fault.ErrAssetNotFound
		}

		tx, _, err := transactionrecord.Packed(transaction).Unpack(mode.IsTesting())
		if nil != err {
			return err
		}

		record, ok := transactionrecord.RecordName(tx)
		if !ok {
			return fault.ErrAssetNotFound
		}
		textAssetIndex, err := assetIndex.MarshalText()
		if nil != err {
			return err
		}

		records[string(textAssetIndex)] = BitmarksRecord{
			Record:     record,
			AssetIndex: assetIndex,
			Data:       tx,
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
