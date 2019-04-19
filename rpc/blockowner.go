// Copyright (c) 2014-2018 Bitmark Inc.
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

type BlockOwner struct {
	log     *logger.L
	limiter *rate.Limiter
}

// get the id for a given block number
// -----------------------------------

type TxIdForBlockArguments struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type TxIdForBlockReply struct {
	TxId merkle.Digest `json:"txId"`
}

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

	header, digest, _, err := blockrecord.ExtractHeader(packedBlock, 0)
	if nil != err {
		return err
	}

	reply.TxId = blockrecord.FoundationTxId(header, digest)

	return nil
}

// Block owner transfer
// --------------------

type BlockOwnerTransferReply struct {
	TxId     merkle.Digest                                   `json:"txId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

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
