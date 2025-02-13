// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitmark

import (
	"bytes"
	"errors"
	"strings"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/ratelimit"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"golang.org/x/time/rate"
)

const (
	rateLimitBitmark = 200
	rateBurstBitmark = 100
)

// Bitmark - type for the RPC
type Bitmark struct {
	Log              *logger.L
	Limiter          *rate.Limiter
	IsNormalMode     func(mode.Mode) bool
	IsTestingChain   func() bool
	Rsvr             reservoir.Reservoir
	PoolTransactions storage.Handle
	PoolAssets       storage.Handle
	PoolOwnerTxIndex storage.Handle
	PoolOwnerData    storage.Handle
	ReadOnly         bool
}

// TransferReply - result from transfer RPC
type TransferReply struct {
	TxId      merkle.Digest                                   `json:"txId"`
	BitmarkId merkle.Digest                                   `json:"bitmarkId"`
	PayId     pay.PayId                                       `json:"payId"`
	Payments  map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

func New(log *logger.L,
	pools reservoir.Handles,
	isNormalMode func(mode.Mode) bool,
	isTestingChain func() bool,
	rsvr reservoir.Reservoir,
	readOnly bool,
) *Bitmark {
	return &Bitmark{
		Log:              log,
		Limiter:          rate.NewLimiter(rateLimitBitmark, rateBurstBitmark),
		IsNormalMode:     isNormalMode,
		IsTestingChain:   isTestingChain,
		Rsvr:             rsvr,
		PoolTransactions: pools.Transactions,
		PoolAssets:       pools.Assets,
		PoolOwnerTxIndex: pools.OwnerTxIndex,
		PoolOwnerData:    pools.OwnerData,
		ReadOnly:         readOnly,
	}
}

// Transfer - transfer a bitmark
func (bitmark *Bitmark) Transfer(arguments *transactionrecord.BitmarkTransferCountersigned, reply *TransferReply) error {
	if err := ratelimit.Limit(bitmark.Limiter); err != nil {
		return err
	}
	if bitmark.ReadOnly {
		return fault.NotAvailableInReadOnlyMode
	}

	log := bitmark.Log
	transfer := transactionrecord.BitmarkTransfer(arguments)

	log.Infof("Bitmark.Transfer: %+v", transfer)

	if arguments == nil || arguments.Owner == nil {
		return fault.InvalidItem
	}

	if !bitmark.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	if arguments.Owner.IsTesting() != bitmark.IsTestingChain() {
		return fault.WrongNetworkForPublicKey
	}

	// for unratified transfers
	if len(arguments.Countersignature) == 0 {
		transfer = &transactionrecord.BitmarkTransferUnratified{
			Link:      arguments.Link,
			Escrow:    arguments.Escrow,
			Owner:     arguments.Owner,
			Signature: arguments.Signature,
		}
	}

	// save transfer/check for duplicate
	stored, duplicate, err := bitmark.Rsvr.StoreTransfer(transfer)

	if err != nil {
		return err
	}

	// only first result needs to be considered
	payId := stored.Id
	txId := stored.TxId
	bitmarkId := stored.IssueTxId
	packedTransfer := stored.Packed

	log.Debugf("id: %v", txId)
	reply.TxId = txId
	reply.BitmarkId = bitmarkId
	reply.PayId = payId
	reply.Payments = make(map[string]transactionrecord.PaymentAlternative)

	for _, payment := range stored.Payments {
		c := payment[0].Currency.String()
		reply.Payments[c] = payment
	}

	// announce transaction block to other peers
	if !duplicate {
		messagebus.Bus.Broadcast.Send("transfer", packedTransfer)
	}

	return nil
}

// Trace the history of a property
// -------------------------------

const (
	maximumProvenanceCount = 100
)

// ProvenanceArguments - arguments for provenance RPC
type ProvenanceArguments struct {
	TxId  merkle.Digest `json:"txId"`
	Count int           `json:"count"`
}

// ProvenanceRecord - can be any of the transaction records
type ProvenanceRecord struct {
	Record  string      `json:"record"`
	IsOwner bool        `json:"isOwner"`
	TxId    interface{} `json:"txId,omitempty"`
	InBlock uint64      `json:"inBlock,string"`
	AssetId interface{} `json:"assetId,omitempty"`
	Data    interface{} `json:"data"`
}

// ProvenanceReply - results from provenance RPC
type ProvenanceReply struct {
	Data []ProvenanceRecord `json:"data"`
}

// Provenance - list the provenance from s transaction id
func (bitmark *Bitmark) Provenance(arguments *ProvenanceArguments, reply *ProvenanceReply) error {

	if err := ratelimit.LimitN(bitmark.Limiter, arguments.Count, maximumProvenanceCount); err != nil {
		return err
	}

	log := bitmark.Log

	log.Infof("Bitmark.Provenance: %+v", arguments)

	count := arguments.Count
	id := arguments.TxId

	provenance := make([]ProvenanceRecord, 0, count)

loop:
	for i := 0; i < count; i += 1 {

		inBlock, packed := bitmark.PoolTransactions.GetNB(id[:])
		if packed == nil {
			break loop
		}

		transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
		if err != nil {
			break loop
		}

		record, _ := transactionrecord.RecordName(transaction)
		h := ProvenanceRecord{
			Record:  record,
			IsOwner: false,
			TxId:    id,
			InBlock: inBlock,
			AssetId: nil,
			Data:    transaction,
		}

		switch tx := transaction.(type) {

		case *transactionrecord.OldBaseData:
			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id, bitmark.PoolOwnerTxIndex)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BlockFoundation:
			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id, bitmark.PoolOwnerTxIndex)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BitmarkIssue:
			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id, bitmark.PoolOwnerTxIndex)
			}
			provenance = append(provenance, h)

			inBlock, packedAsset := bitmark.PoolAssets.GetNB(tx.AssetId[:])
			if packedAsset == nil {
				break loop
			}
			assetTx, _, err := transactionrecord.Packed(packedAsset).Unpack(mode.IsTesting())
			if err != nil {
				break loop
			}

			record, _ := transactionrecord.RecordName(assetTx)
			h = ProvenanceRecord{
				Record:  record,
				IsOwner: false,
				TxId:    nil,
				InBlock: inBlock,
				AssetId: tx.AssetId,
				Data:    assetTx,
			}
			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned, *transactionrecord.BlockOwnerTransfer:
			tr := tx.(transactionrecord.BitmarkTransfer)

			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tr.GetOwner(), id, bitmark.PoolOwnerTxIndex)
			}

			provenance = append(provenance, h)
			id = tr.GetLink()

		case *transactionrecord.BitmarkShare:
			h.IsOwner = true // share terminates a provenance chain so will always be owner
			provenance = append(provenance, h)
			id = tx.Link

		default:
			break loop
		}
	}

	reply.Data = provenance

	return nil
}

// Trace the full history of a bitmarkId
// -------------------------------------

// FullProvenanceArguments - arguments for provenance RPC
type FullProvenanceArguments struct {
	BitmarkId merkle.Digest `json:"bitmarkId"`
}

// FullProvenanceRecord - can be any of the transaction records
type FullProvenanceRecord struct {
	Record   string      `json:"record"`
	IsOwner  bool        `json:"isOwner,omitempty"`
	TxId     interface{} `json:"txId,omitempty"`
	InBlock  uint64      `json:"inBlock,string"`
	AssetId  interface{} `json:"assetId,omitempty"`
	Data     interface{} `json:"data"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// FullProvenanceReply - results from provenance RPC
type FullProvenanceReply struct {
	Data []FullProvenanceRecord `json:"data"`
}

var errDone = errors.New("scanning is done")

// FullProvenance - list the provenance from s transaction id
func (bitmark *Bitmark) FullProvenance(arguments *FullProvenanceArguments, reply *FullProvenanceReply) error {

	if err := ratelimit.LimitN(bitmark.Limiter, 1, 1); err != nil {
		return err
	}

	log := bitmark.Log

	log.Infof("Bitmark.FullProvenance: %+v", arguments)

	bitmarkId := arguments.BitmarkId

	// data: 00 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ asset id
	// data: 01 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ owned BN
	// data: 02 ⧺ transfer BN ⧺ issue txId ⧺ issue BN ⧺ asset id
	cursor := bitmark.PoolOwnerData.NewFetchCursor()

	id := merkle.Digest{}

	err := cursor.Map(func(key []byte, value []byte) error {
		if bytes.Equal(bitmarkId[:], value[9:9+32]) {
			copy(id[:], key)
			return errDone
		}
		return nil
	})

	if err == nil {
		return fault.TransactionIsNotAnIssue
	} else if errDone != err {
		return err
	}

	provenance := make([]FullProvenanceRecord, 0, 10000)

loop:
	for i := 0; true; i += 1 {

		inBlock, packed := bitmark.PoolTransactions.GetNB(id[:])
		if packed == nil {
			break loop
		}

		transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
		if err != nil {
			break loop
		}

		record, _ := transactionrecord.RecordName(transaction)
		h := FullProvenanceRecord{
			Record:  record,
			IsOwner: false,
			TxId:    id,
			InBlock: inBlock,
			AssetId: nil,
			Data:    transaction,
		}

		switch tx := transaction.(type) {

		case *transactionrecord.OldBaseData:
			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id, bitmark.PoolOwnerTxIndex)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BlockFoundation:
			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id, bitmark.PoolOwnerTxIndex)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BitmarkIssue:
			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id, bitmark.PoolOwnerTxIndex)
			}
			provenance = append(provenance, h)

			inBlock, packedAsset := bitmark.PoolAssets.GetNB(tx.AssetId[:])
			if packedAsset == nil {
				break loop
			}
			assetTx, _, err := transactionrecord.Packed(packedAsset).Unpack(mode.IsTesting())
			if err != nil {
				break loop
			}

			meta := strings.Split(assetTx.(*transactionrecord.AssetData).Metadata, "\u0000")
			metamap := make(map[string]string)
			for i := 0; i < len(meta); i += 2 {
				metamap[meta[i]] = meta[i+1]
			}

			record, _ := transactionrecord.RecordName(assetTx)
			h = FullProvenanceRecord{
				Record:   record,
				IsOwner:  false,
				TxId:     nil,
				InBlock:  inBlock,
				AssetId:  tx.AssetId,
				Data:     assetTx,
				Metadata: metamap,
			}
			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned, *transactionrecord.BlockOwnerTransfer:
			tr := tx.(transactionrecord.BitmarkTransfer)

			if i == 0 {
				h.IsOwner = ownership.CurrentlyOwns(nil, tr.GetOwner(), id, bitmark.PoolOwnerTxIndex)
			}

			provenance = append(provenance, h)
			id = tr.GetLink()

		case *transactionrecord.BitmarkShare:
			h.IsOwner = true // share terminates a provenance chain so will always be owner
			provenance = append(provenance, h)
			id = tx.Link

		default:
			break loop
		}
	}

	reply.Data = provenance

	return nil
}
