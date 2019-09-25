// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
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
	log     *logger.L
	limiter *rate.Limiter
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

	if err := rateLimit(bitmark.limiter); nil != err {
		return err
	}

	log := bitmark.log

	log.Infof("BlockOwner.TxIdForBlock: %+v", info)

	blockNumberKey := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumberKey, info.BlockNumber)
	packedBlock := storage.Pool.Blocks.Get(blockNumberKey)
	if nil == packedBlock {
		return fault.ErrBlockNotFound
	}

	header, digest, _, err := blockrecord.ExtractHeader(packedBlock, 0, false)
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

	if err := rateLimit(bitmark.limiter); nil != err {
		return err
	}

	log := bitmark.log

	log.Infof("BlockOwner.Transfer: %+v", transfer)

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
