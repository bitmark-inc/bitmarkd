// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockowner

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/blockrecord"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/ratelimit"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"golang.org/x/time/rate"
)

// Block Owner
// -----------

const (
	rateLimitBlockOwner = 200
	rateBurstBlockOwner = 100
)

// BlockOwner - the type of the RPC
type BlockOwner struct {
	Log            *logger.L
	Limiter        *rate.Limiter
	Pool           storage.Handle
	Br             blockrecord.Record
	IsNormalMode   func(mode.Mode) bool
	IsTestingChain func() bool
	Rsvr           reservoir.Reservoir
	ReadOnly       bool
}

// TxIDForBlockArguments - get the id for a given block number
type TxIDForBlockArguments struct {
	BlockNumber uint64 `json:"blockNumber"`
}

// TxIDForBlockReply - results for block id
type TxIDForBlockReply struct {
	TxId merkle.Digest `json:"txId"`
}

func New(log *logger.L,
	pools reservoir.Handles,
	isNormalMode func(mode.Mode) bool,
	isTestingChain func() bool,
	rsvr reservoir.Reservoir,
	br blockrecord.Record,
	readOnly bool,
) *BlockOwner {
	return &BlockOwner{
		Log:            log,
		Limiter:        rate.NewLimiter(rateLimitBlockOwner, rateBurstBlockOwner),
		Pool:           pools.Blocks,
		Br:             br,
		IsNormalMode:   isNormalMode,
		IsTestingChain: isTestingChain,
		Rsvr:           rsvr,
		ReadOnly:       readOnly,
	}
}

// TxIDForBlock - RPC to get transaction id for block ownership record
func (bitmark *BlockOwner) TxIDForBlock(info *TxIDForBlockArguments, reply *TxIDForBlockReply) error {

	if err := ratelimit.Limit(bitmark.Limiter); err != nil {
		return err
	}

	log := bitmark.Log

	log.Infof("BlockOwner.TxIDForBlock: %+v", info)

	if bitmark.Pool == nil {
		return fault.DatabaseIsNotSet
	}

	blockNumberKey := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumberKey, info.BlockNumber)
	packedBlock := bitmark.Pool.Get(blockNumberKey)
	if packedBlock == nil {
		return fault.BlockNotFound
	}

	header, digest, _, err := bitmark.Br.ExtractHeader(packedBlock, 0, false)
	if err != nil {
		return err
	}

	reply.TxId = blockrecord.FoundationTxId(header.Number, digest)

	return nil
}

// Block owner transfer
// --------------------

// TransferReply - results of transferring block ownership
type TransferReply struct {
	TxId     merkle.Digest                                   `json:"txId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Transfer - transfer the ownership of a block to new account and/or
// payment addresses
func (bitmark *BlockOwner) Transfer(transfer *transactionrecord.BlockOwnerTransfer, reply *TransferReply) error {

	if err := ratelimit.Limit(bitmark.Limiter); err != nil {
		return err
	}
	if bitmark.ReadOnly {
		return fault.NotAvailableInReadOnlyMode
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
	if err != nil {
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
