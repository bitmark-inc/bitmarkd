// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/payment"
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
	TxId     merkle.Digest `json:"txId"`
	PayId    payment.PayId `json:"payId"`
	Payments []*transactionrecord.Payment
	//PaymentAlternatives []block.MinerAddress `json:"paymentAlternatives"`// ***** FIX THIS: where to get addresses?
}

func (bitmark *Bitmark) Transfer(arguments *transactionrecord.BitmarkTransfer, reply *BitmarkTransferReply) error {

	log := bitmark.log

	log.Infof("Bitmark.Transfer: %v", arguments)

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	txId, packedTransfer, previousTransfer, ownerData, err := block.VerifyTransfer(arguments)
	if nil != err {
		return err
	}

	log.Infof("packed transfer: %x", packedTransfer)
	log.Infof("id: %v", txId)
	log.Infof("ownerData: %x", ownerData)

	// get block number of transfer and issue; see: storage/doc.go to determine offsets
	const transferBlockNumberOffset = merkle.DigestLength
	const issueBlockNumberOffset = 8 + 2*merkle.DigestLength

	tKey := ownerData[transferBlockNumberOffset : transferBlockNumberOffset+8]
	iKey := ownerData[issueBlockNumberOffset : issueBlockNumberOffset+8]

	log.Infof("iKey: %x  tKey: %x", iKey, tKey)

	// block owner (from issue) payment
	// 0: the issue owner
	// 1: block miner (TO DO)
	// 2: transfer payment (optional)
	payments := make([]*transactionrecord.Payment, 1, 3)
	payments[0] = block.GetPayment(iKey)

	// last transfer payment if there is one
	for _, x := range tKey {
		if 0 != x {
			p := block.GetPayment(tKey)
			if p.Currency == payments[0].Currency && p.Address == payments[0].Address {
				payments[0].Amount += p.Amount
			} else {
				payments = append(payments, p)
			}
			break
		}
	}

	// optional payment record (if previous record was transfer and contains such)
	if nil != previousTransfer && nil != previousTransfer.Payment {
		payments = append(payments, previousTransfer.Payment)
	}

	// get payment info
	reply.TxId = txId
	reply.PayId, _, _ = payment.Store(currency.Bitcoin, packedTransfer, 1, false)
	reply.Payments = payments

	// announce transaction block to other peers
	messagebus.Bus.Broadcast.Send("transfer", packedTransfer)

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

		case *transactionrecord.BitmarkTransfer:

			if 0 == i {
				dKey := append(tx.Owner.Bytes(), id[:]...)
				if nil != storage.Pool.OwnerDigest.Get(dKey) {
					h.IsOwner = true
				}
			}

			provenance = append(provenance, h)
			id = tx.Link

		default:
			break loop
		}
	}

	reply.Data = provenance

	return nil
}
