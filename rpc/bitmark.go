// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Bitmark
// -------

type Bitmark struct {
	log *logger.L
}

// Bitmark transfer
// ----------------

type BitmarkTransferReply struct {
	TxId     merkle.Digest                                   `json:"txId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

func (bitmark *Bitmark) Transfer(transfer *transactionrecord.BitmarkTransferCountersigned, reply *BitmarkTransferReply) error {

	log := bitmark.log

	log.Infof("Bitmark.Transfer: %v", transfer)

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	if transfer.Owner.IsTesting() != mode.IsTesting() {
		return fault.ErrWrongNetworkForPublicKey
	}

	// save transfer/check for duplicate
	stored, duplicate, err := reservoir.StoreTransfer(transfer)
	if nil != err {
		return err
	}

	// only first result needs to be considered
	payId := stored.Id
	txId := stored.TxId
	packedTransfer := stored.Packed

	log.Infof("id: %v", txId)
	reply.TxId = txId
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

type ProvenanceArguments struct {
	TxId  merkle.Digest `json:"txId"`
	Count int           `json:"count"`
}

// can be any of the transaction records
type ProvenanceRecord struct {
	Record     string      `json:"record"`
	IsOwner    bool        `json:"isOwner"`
	TxId       interface{} `json:"txId,omitempty"`
	AssetIndex interface{} `json:"index,omitempty"`
	Data       interface{} `json:"data"`
}

type ProvenanceReply struct {
	Data []ProvenanceRecord `json:"data"`
}

func (bitmark *Bitmark) Provenance(arguments *ProvenanceArguments, reply *ProvenanceReply) error {
	log := bitmark.log

	log.Infof("Bitmark.Provenance: %v", arguments)

	count := arguments.Count
	id := arguments.TxId

	if count <= 0 {
		return fault.ErrInvalidCount
	}

	provenance := make([]ProvenanceRecord, 0, count)

loop:
	for i := 0; i < count; i += 1 {

		packed := storage.Pool.Transactions.Get(id[:])
		if nil == packed {
			break loop
		}

		transaction, _, err := transactionrecord.Packed(packed).Unpack()
		if nil != err {
			break loop
		}

		record, _ := transactionrecord.RecordName(transaction)
		h := ProvenanceRecord{
			Record:     record,
			IsOwner:    false,
			TxId:       id,
			AssetIndex: nil,
			Data:       transaction,
		}

		switch tx := transaction.(type) {

		case *transactionrecord.BitmarkIssue:

			if 0 == i {
				dKey := append(tx.Owner.Bytes(), id[:]...)
				if nil != storage.Pool.OwnerDigest.Get(dKey) {
					h.IsOwner = true
				}
			}

			provenance = append(provenance, h)

			packedAsset := storage.Pool.Assets.Get(tx.AssetIndex[:])
			if nil == packedAsset {
				break loop
			}
			assetTx, _, err := transactionrecord.Packed(packedAsset).Unpack()
			if nil != err {
				break loop
			}

			record, _ := transactionrecord.RecordName(assetTx)
			h := ProvenanceRecord{
				Record:     record,
				IsOwner:    false,
				TxId:       nil,
				AssetIndex: tx.AssetIndex,
				Data:       assetTx,
			}
			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned:
			tr := tx.(transactionrecord.BitmarkTransfer)

			if 0 == i {
				dKey := append(tr.GetOwner().Bytes(), id[:]...)
				if nil != storage.Pool.OwnerDigest.Get(dKey) {
					h.IsOwner = true
				}
			}

			provenance = append(provenance, h)
			id = tr.GetLink()

		default:
			break loop
		}
	}

	reply.Data = provenance

	return nil
}
