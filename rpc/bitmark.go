// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/logger"
)

// Bitmark
// -------

type Bitmark struct {
	log *logger.L
}

// Bitmark issue
// -------------

type BitmarkIssueReply struct {
	TxId           transaction.Link     `json:"txid"`
	PaymentAddress []block.MinerAddress `json:"paymentAddress"`
	Duplicate      bool                 `json:"duplicate"`
	Err            string               `json:"error,omitempty"`
}

func (bitmark *Bitmark) Issue(arguments *transaction.BitmarkIssue, reply *BitmarkIssueReply) error {

	log := bitmark.log

	log.Infof("Bitmark.Issue: %v", arguments)

	packedIssue, err := arguments.Pack(arguments.Owner)
	if nil != err {
		return err
	}

	// check record
	id, exists := packedIssue.Exists()

	// announce transaction to system
	if !exists {
		messagebus.Send("", packedIssue)
	}

	log.Infof("Bitmark.Issue exists: %v", exists)

	// set up reply
	reply.TxId = id
	reply.PaymentAddress = payment.PaymentAddresses()
	reply.Duplicate = exists

	return nil
}

// Bitmark transfer
// ----------------

type BitmarkTransferReply struct {
	TxId           transaction.Link     `json:"txid"`
	PaymentAddress []block.MinerAddress `json:"paymentAddress"`
	Duplicate      bool                 `json:"duplicate"`
	Err            string               `json:"error,omitempty"`
}

func (bitmark *Bitmark) Transfer(arguments *transaction.BitmarkTransfer, reply *BitmarkTransferReply) error {

	log := bitmark.log

	log.Infof("Bitmark.Transfer: %v", arguments)

	state, packedTransaction, found := arguments.Link.Read()
	if !found {
		return fault.ErrLinkNotFound
	}

	// predecessor must already be confirmed
	if state != transaction.ConfirmedTransaction {
		return fault.ErrLinksToUnconfirmedTransaction
	}

	trans, err := packedTransaction.Unpack()
	if nil != err {
		return err
	}

	// extract address and exclude impossible chain links
	var address *transaction.Address
	switch trans.(type) {
	case *transaction.BitmarkIssue:
		address = trans.(*transaction.BitmarkIssue).Owner
		// predecessor must be the current issuer
		if !arguments.Link.IsOwner(address) {
			return fault.ErrNotCurrentOwner
		}

	case *transaction.BitmarkTransfer:
		address = trans.(*transaction.BitmarkTransfer).Owner
		// predecessor must be the current owner
		if !arguments.Link.IsOwner(address) {
			return fault.ErrNotCurrentOwner
		}
	default:
		return fault.ErrInvalidTransactionChain
	}

	packedTransfer, err := arguments.Pack(address)
	if nil != err {
		return err
	}

	// check record
	id, exists := packedTransfer.Exists()

	reply.Duplicate = exists
	reply.TxId = id
	reply.PaymentAddress = payment.PaymentAddresses()

	// announce transaction to system
	if !exists {
		messagebus.Send("", packedTransfer)
	}

	log.Infof("Bitmark.Transfer exists: %v", exists)
	return nil
}

// Trace the history of a property
// -------------------------------

type ProvenanceArguments struct {
	TxId  transaction.Link `json:"txid"`
	Count int              `json:"count"`
}

// can be any of the transaction records
type ProvenanceRecord struct {
	Record string            `json:"record"`
	TxId   transaction.Link  `json:"txid"`
	State  transaction.State `json:"state"`
	Data   interface{}       `json:"data"`
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
		state, data, found := id.Read()
		if !found {
			break loop
		}

		tx, err := data.Unpack()
		if nil != err {
			break loop
		}

		link := id // save for later

		record := "*unknown*"
		done := false

		switch tx.(type) {
		case *transaction.AssetData:
			record = "AssetData"
			done = true

		case *transaction.BitmarkIssue:
			record = "BitmarkIssue"
			_, id, found = tx.(*transaction.BitmarkIssue).AssetIndex.Read()
			if !found {
				done = true
			}
			//id = tx.(*transaction.BitmarkIssue).Link

		case *transaction.BitmarkTransfer:
			record = "BitmarkTransfer"
			id = tx.(*transaction.BitmarkTransfer).Link

		default:
			break loop
		}

		h := ProvenanceRecord{
			Record: record,
			TxId:   link,
			State:  state,
			Data:   tx,
		}
		provenance = append(provenance, h)
		if done {
			break loop
		}
	}
	reply.Data = provenance

	return nil
}
