// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
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

func (bitmark *Bitmark) Transfer(arguments *transactionrecord.BitmarkTransferCountersigned, reply *BitmarkTransferReply) error {

	log := bitmark.log
	transfer := transactionrecord.BitmarkTransfer(arguments)

	log.Infof("Bitmark.Transfer: %+v", transfer)

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	if arguments.Owner.IsTesting() != mode.IsTesting() {
		return fault.ErrWrongNetworkForPublicKey
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
	stored, duplicate, err := reservoir.StoreTransfer(transfer)
	if nil != err {
		return err
	}

	// only first result needs to be considered
	payId := stored.Id
	txId := stored.TxId
	packedTransfer := stored.Packed

	log.Debugf("id: %v", txId)
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

const (
	maximumProvenanceCount = 200
)

type ProvenanceArguments struct {
	TxId  merkle.Digest `json:"txId"`
	Count int           `json:"count"`
}

// can be any of the transaction records
type ProvenanceRecord struct {
	Record     string      `json:"record"`
	IsOwner    bool        `json:"isOwner"`
	TxId       interface{} `json:"txId,omitempty"`
	InBlock    uint64      `json:"inBlock"`
	AssetIndex interface{} `json:"assetIndex,omitempty"`
	Data       interface{} `json:"data"`
}

type ProvenanceReply struct {
	Data []ProvenanceRecord `json:"data"`
}

func (bitmark *Bitmark) Provenance(arguments *ProvenanceArguments, reply *ProvenanceReply) error {
	log := bitmark.log

	log.Infof("Bitmark.Provenance: %+v", arguments)

	count := arguments.Count
	id := arguments.TxId

	if count <= 0 {
		return fault.ErrInvalidCount
	}
	if count > maximumProvenanceCount {
		count = maximumProvenanceCount
	}

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
			Record:     record,
			IsOwner:    false,
			TxId:       id,
			InBlock:    inBlock,
			AssetIndex: nil,
			Data:       transaction,
		}

		switch tx := transaction.(type) {

		case *transactionrecord.OldBaseData:
			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(tx.Owner, id)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BlockFoundation:
			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(tx.Owner, id)
			}

			provenance = append(provenance, h)
			break loop

		case *transactionrecord.BitmarkIssue:
			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(tx.Owner, id)
			}
			provenance = append(provenance, h)

			_, packedAsset := storage.Pool.Assets.GetNB(tx.AssetIndex[:])
			if nil == packedAsset {
				break loop
			}
			assetTx, _, err := transactionrecord.Packed(packedAsset).Unpack(mode.IsTesting())
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

		case *transactionrecord.BitmarkTransferUnratified, *transactionrecord.BitmarkTransferCountersigned, *transactionrecord.BlockOwnerTransfer:
			tr := tx.(transactionrecord.BitmarkTransfer)

			if 0 == i {
				h.IsOwner = ownership.CurrentlyOwns(tr.GetOwner(), id)
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
