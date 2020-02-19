// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/ownership"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Bitmark - type for the RPC
type Bitmark struct {
	Log            *logger.L
	Limiter        *rate.Limiter
	IsNormalMode   func(mode.Mode) bool
	IsTestingChain func() bool
	Rsvr           reservoir.Reservoir
}

// BitmarkTransferReply - result from transfer RPC
type BitmarkTransferReply struct {
	TxId      merkle.Digest                                   `json:"txId"`
	BitmarkId merkle.Digest                                   `json:"bitmarkId"`
	PayId     pay.PayId                                       `json:"payId"`
	Payments  map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Transfer - transfer a bitmark
func (bitmark *Bitmark) Transfer(arguments *transactionrecord.BitmarkTransferCountersigned, reply *BitmarkTransferReply) error {
	if err := rateLimit(bitmark.Limiter); nil != err {
		return err
	}

	log := bitmark.Log
	transfer := transactionrecord.BitmarkTransfer(arguments)

	log.Infof("Bitmark.Transfer: %+v", transfer)

	if nil == arguments || nil == arguments.Owner {
		return fault.InvalidItem
	}

	if !bitmark.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	if arguments.Owner.IsTesting() != bitmark.IsTestingChain() {
		return fault.WrongNetworkForPublicKey
	}

	// for unratified transfers
	if 0 == len(arguments.Countersignature) {
		transfer = &transactionrecord.BitmarkTransferUnratified{
			Link:      arguments.Link,
			Escrow:    arguments.Escrow,
			Owner:     arguments.Owner,
			Signature: arguments.Signature,
		}
	}

	// save transfer/check for duplicate
	stored, duplicate, err := bitmark.Rsvr.StoreTransfer(transfer)

	if nil != err {
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
		messagebus.Bus.P2P.Send("transfer", packedTransfer)
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
	InBlock uint64      `json:"inBlock"`
	AssetId interface{} `json:"assetId,omitempty"`
	Data    interface{} `json:"data"`
}

// ProvenanceReply - results from provenance RPC
type ProvenanceReply struct {
	Data []ProvenanceRecord `json:"data"`
}

// Provenance - list the provenance from s transaction id
func (bitmark *Bitmark) Provenance(arguments *ProvenanceArguments, reply *ProvenanceReply) error {

	if err := rateLimitN(bitmark.Limiter, arguments.Count, maximumProvenanceCount); nil != err {
		return err
	}

	log := bitmark.Log

	log.Infof("Bitmark.Provenance: %+v", arguments)

	count := arguments.Count
	id := arguments.TxId

	provenance := make([]ProvenanceRecord, 0, count)

loop:
	for i := 0; i < count; i += 1 {

		inBlock, packed := storage.Pool.Transactions.GetNB(id[:])
		if nil == packed {
			break loop
		}

		transaction, _, err := transactionrecord.Packed(packed).Unpack(mode.IsTesting())
		if nil != err {
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
			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BlockFoundation:
			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BitmarkIssue:
			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(nil, tx.Owner, id)
			}
			provenance = append(provenance, h)

			inBlock, packedAsset := storage.Pool.Assets.GetNB(tx.AssetId[:])
			if nil == packedAsset {
				break loop
			}
			assetTx, _, err := transactionrecord.Packed(packedAsset).Unpack(mode.IsTesting())
			if nil != err {
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

			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(nil, tr.GetOwner(), id)
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
