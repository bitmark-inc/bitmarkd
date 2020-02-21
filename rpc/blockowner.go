// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/binary"

	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
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

// Block Owner
// -----------

// BlockOwner - the type of the RPC
type BlockOwner struct {
	Log            *logger.L
	Limiter        *rate.Limiter
	Pool           storage.Handle
	Br             blockrecord.Record
	IsNormalMode   func(mode.Mode) bool
	IsTestingChain func() bool
	Rsvr           reservoir.Reservoir
}

// TxIdForBlockArguments - get the id for a given block number
type TxIdForBlockArguments struct {
	BlockNumber uint64 `json:"blockNumber"`
}

// TxIdForBlockReply - results for block id
type TxIdForBlockReply struct {
	TxId merkle.Digest `json:"txId"`
}

// TxIdForBlock - RPC to get transaction id for block ownership record
func (bitmark *BlockOwner) TxIdForBlock(info *TxIdForBlockArguments, reply *TxIdForBlockReply) error {

	if err := rateLimit(bitmark.Limiter); nil != err {
		return err
	}

	log := bitmark.Log

	log.Infof("BlockOwner.TxIdForBlock: %+v", info)

	blockNumberKey := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumberKey, info.BlockNumber)
	packedBlock := bitmark.Pool.Get(blockNumberKey)
	if nil == packedBlock {
		return fault.BlockNotFound
	}

	header, digest, _, err := bitmark.Br.ExtractHeader(packedBlock, 0, false)
	if nil != err {
		return err
	}

	reply.TxId = blockrecord.FoundationTxId(header.Number, digest)

	return nil
}

// Block owner transfer
// --------------------

// BlockOwnerTransferReply - results of transferring block ownership
type BlockOwnerTransferReply struct {
	TxId     merkle.Digest                                   `json:"txId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Transfer - transfer the ownership of a block to new account and/or
// payment addresses
func (bitmark *BlockOwner) Transfer(transfer *transactionrecord.BlockOwnerTransfer, reply *BlockOwnerTransferReply) error {

	if err := rateLimit(bitmark.Limiter); nil != err {
		return err
	}

	log := bitmark.Log

	log.Infof("BlockOwner.Transfer: %+v", transfer)

	if !bitmark.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	if transfer.Owner.IsTesting() != bitmark.IsTestingChain() {
		return fault.WrongNetworkForPublicKey
	}

	// save transfer/check for duplicate
	stored, duplicate, err := bitmark.Rsvr.StoreTransfer(transfer)
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
		messagebus.Bus.P2P.Send("transfer", packedTransfer)
	}

	return nil
}
