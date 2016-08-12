// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/account"
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
	TxId     merkle.Digest `json:"txid"`
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

	// find the current owner via the link
	previousPacked := storage.Pool.Transactions.Get(arguments.Link[:])
	if nil == previousPacked {
		return fault.ErrLinkToInvalidOrUnconfirmedTransaction
	}

	previousTransaction, _, err := transactionrecord.Packed(previousPacked).Unpack()
	if nil != err {
		return err
	}

	var currentOwner *account.Account
	var previousTransfer *transactionrecord.BitmarkTransfer

	switch previousTransaction.(type) {
	case *transactionrecord.BitmarkIssue:
		issue := previousTransaction.(*transactionrecord.BitmarkIssue)
		currentOwner = issue.Owner

	case *transactionrecord.BitmarkTransfer:
		transfer := previousTransaction.(*transactionrecord.BitmarkTransfer)
		currentOwner = transfer.Owner
		previousTransfer = transfer

	default:
		return fault.ErrLinkToInvalidOrUnconfirmedTransaction
	}

	// pack transfer and check signature
	packedTransfer, err := arguments.Pack(currentOwner)
	if nil != err {
		return err
	}

	// transfer identifier and check for duplicate
	txId := packedTransfer.MakeLink()
	key := txId[:]
	if storage.Pool.Transactions.Has(key) || storage.Pool.VerifiedTransactions.Has(key) {
		return fault.ErrTransactionAlreadyExists
	}

	log.Infof("packed transfer: %x", packedTransfer)
	log.Infof("id: %v", txId)

	// get count for current owner record
	// to make sure that the record has not already been transferred
	dKey := append(currentOwner.Bytes(), arguments.Link[:]...)
	log.Infof("dKey: %x", dKey)
	dCount := storage.Pool.OwnerDigest.Get(dKey)
	if nil == dCount {
		return fault.ErrDoubleTransferAttempt
	}
	log.Infof("dCount: %x", dCount)

	// get ownership data
	oKey := append(currentOwner.Bytes(), dCount...)
	log.Infof("oKey: %x", oKey)
	ownerData := storage.Pool.Ownership.Get(oKey)
	if nil == ownerData {
		return fault.ErrDoubleTransferAttempt
	}
	log.Infof("ownerData: %x", ownerData)

	// get block number of issue
	bKey := ownerData[2*merkle.DigestLength+transactionrecord.AssetIndexLength:]
	if 8 != len(bKey) {
		log.Criticalf("expected 8 byte block number but got: %d bytes", len(bKey))
		fault.Panicf("expected 8 byte block number but got: %d bytes", len(bKey))
	}
	log.Infof("bKey: %x", bKey)

	blockOwnerData := storage.Pool.BlockOwners.Get(bKey)
	if nil == blockOwnerData {
		return fault.ErrDoubleTransferAttempt
	}
	log.Infof("blockOwnerData: %x", blockOwnerData)

	// block owner (from issue) payment
	// 0: the issue owner
	// 1: block miner (TO DO)
	// 2: transfer payment (optional)
	payments := make([]*transactionrecord.Payment, 1, 3)
	c, err := currency.FromUint64(binary.BigEndian.Uint64(blockOwnerData[:8]))
	if nil != err {
		log.Criticalf("block currency invalid error: %v", err)
		fault.Panicf("block currency invalid error: %v", err)
	}
	payments[0] = &transactionrecord.Payment{
		Currency: c,
		Address:  string(blockOwnerData[8:]),
		Amount:   5000, // ***** FIX THIS: what is the correct value
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
	TxId  merkle.Digest `json:"txid"`
	Count int           `json:"count"`
}

// can be any of the transaction records
type ProvenanceRecord struct {
	Record  string      `json:"record"`
	TxId    interface{} `json:"txid"`
	AssetId interface{} `json:"assetid"`
	Data    interface{} `json:"data"`
}

type ProvenanceReply struct {
	Data []ProvenanceRecord `json:"data"`
}

func (bitmark *Bitmark) Provenance(arguments *ProvenanceArguments, reply *ProvenanceReply) error {
	log := bitmark.log

	log.Infof("Bitmark.Provenance: %v", arguments)

	count := arguments.Count
	id := arguments.TxId

	provenance := make([]ProvenanceRecord, 0, count)

loop:
	for i := 0; i < count; i += 1 {

		packed := storage.Pool.Transactions.Get(id[:])
		if nil == packed {
			break loop
		}

		tx, _, err := transactionrecord.Packed(packed).Unpack()
		if nil != err {
			break loop
		}

		record, _ := transactionrecord.RecordName(tx)
		h := ProvenanceRecord{
			Record:  record,
			TxId:    id,
			AssetId: nil,
			Data:    tx,
		}
		provenance = append(provenance, h)

		switch tx.(type) {

		case *transactionrecord.BitmarkIssue:
			issue := tx.(*transactionrecord.BitmarkIssue)
			if i >= count {
				break loop
			}
			asset := storage.Pool.Assets.Get(issue.AssetIndex[:])
			if nil == asset {
				break loop
			}
			tx, _, err := transactionrecord.Packed(asset).Unpack()
			if nil != err {
				break loop
			}

			record, _ := transactionrecord.RecordName(tx)
			h := ProvenanceRecord{
				Record:  record,
				TxId:    nil,
				AssetId: issue.AssetIndex,
				Data:    tx,
			}
			provenance = append(provenance, h)
			break loop

			//id = tx.(*transaction.BitmarkIssue).Link

		case *transactionrecord.BitmarkTransfer:
			id = tx.(*transactionrecord.BitmarkTransfer).Link

		default:
			break loop
		}
	}

	reply.Data = provenance

	return nil
}
