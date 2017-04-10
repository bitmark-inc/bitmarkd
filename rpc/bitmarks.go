// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

// Bitmarks
// --------

type Bitmarks struct {
	log *logger.L
}

const (
	maximumIssues = 100
)

// Bitmarks issue
// --------------

type IssueStatus struct {
	TxId merkle.Digest `json:"txId"`
}

type BitmarksIssueReply struct {
	Issues     []IssueStatus      `json:"issues"`
	PayId      pay.PayId          `json:"payId"`
	PayNonce   reservoir.PayNonce `json:"payNonce"`
	Difficulty string             `json:"difficulty"`
	//PaymentAlternatives []block.MinerAddress `json:"paymentAlternatives"`// ***** FIX THIS: where to get addresses?
}

// Bitmarks create
// --------------

type CreateArguments struct {
	Assets []*transactionrecord.AssetData    `json:"assets"`
	Issues []*transactionrecord.BitmarkIssue `json:"issues"`
}

type CreateReply struct {
	Assets     []AssetStatus      `json:"assets"`
	Issues     []IssueStatus      `json:"issues"`
	PayId      pay.PayId          `json:"payId"`
	PayNonce   reservoir.PayNonce `json:"payNonce"`
	Difficulty string             `json:"difficulty"`
	//PaymentAlternatives []block.MinerAddress `json:"paymentAlternatives"`// ***** FIX THIS: where to get addresses?

}

func (bitmarks *Bitmarks) Create(arguments *CreateArguments, reply *CreateReply) error {

	log := bitmarks.log
	assetCount := len(arguments.Assets)
	issueCount := len(arguments.Issues)

	if assetCount > maximumIssues || issueCount > maximumIssues {
		return fault.ErrTooManyItemsToProcess
	} else if 0 == assetCount && 0 == issueCount {
		return fault.ErrMissingParameters
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	log.Infof("Bitmarks.Create: %v", arguments)

	assetStatus, packedAssets, err := assetRegister(arguments.Assets)
	if nil != err {
		return err
	}

	result := CreateReply{
		Assets: assetStatus,
	}

	packedIssues := []byte{}
	var stored *reservoir.IssueInfo
	duplicate := false
	if issueCount > 0 {
		stored, duplicate, err = reservoir.StoreIssues(arguments.Issues, false)
		if nil != err {
			return err
		}
		packedIssues = stored.Packed
		issueStatus := make([]IssueStatus, len(stored.TxIds))
		for i, txId := range stored.TxIds {
			issueStatus[i].TxId = txId
		}
		result.Issues = issueStatus
	}

	// fail if no data sent
	if 0 == len(assetStatus) && 0 == len(packedIssues) {
		return fault.ErrMissingParameters
	}
	// if data to send
	if 0 != len(packedAssets) {
		// announce transaction block to other peers
		messagebus.Bus.Broadcast.Send("assets", packedAssets)
	}

	if 0 != len(packedIssues) {

		result.PayId = stored.Id
		result.PayNonce = stored.Nonce
		result.Difficulty = stored.Difficulty.GoString()

		// announce transaction block to other peers
		if !duplicate {
			messagebus.Bus.Broadcast.Send("issues", packedIssues, util.ToVarint64(0))
		}
	}

	*reply = result
	return nil
}

// Bitmarks proof
// --------------

type ProofArguments struct {
	PayId pay.PayId `json:"payId"`
	Nonce string    `json:"nonce"`
}

type ProofReply struct {
	Status reservoir.TrackingStatus `json:"status"`
}

func (bitmarks *Bitmarks) Proof(arguments *ProofArguments, reply *ProofReply) error {

	log := bitmarks.log

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	// arbitrary byte size limit
	size := hex.DecodedLen(len(arguments.Nonce))
	if size < 1 || size > payment.NonceLength {
		return fault.ErrInvalidNonce
	}

	log.Infof("proof for pay id: %v", arguments.PayId)
	log.Infof("client nonce: %q", arguments.Nonce)

	nonce := make([]byte, size)
	byteCount, err := hex.Decode(nonce, []byte(arguments.Nonce))
	if nil != err {
		return err
	}
	if byteCount != size {
		return fault.ErrInvalidNonce
	}

	log.Infof("client nonce hex: %x", nonce)

	// announce proof block to other peers
	packed := make([]byte, len(arguments.PayId), len(arguments.PayId)+len(nonce))
	copy(packed, arguments.PayId[:])
	packed = append(packed, nonce...)

	log.Infof("broadcast proof: %x", packed)
	messagebus.Bus.Broadcast.Send("proof", packed)

	// check if proof matches
	reply.Status = reservoir.TryProof(arguments.PayId, nonce)

	return nil
}

// Bitmarks pay
// --------------

type PayArguments struct {
	PayId   pay.PayId `json:"payId"`   // id from the issue/transfer request
	Receipt string    `json:"receipt"` // hex id from payment process
}

type PayReply struct {
	Status reservoir.TrackingStatus `json:"status"`
}

func (bitmarks *Bitmarks) Pay(arguments *PayArguments, reply *PayReply) error {

	log := bitmarks.log

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	// arbitrary byte size limit
	size := hex.DecodedLen(len(arguments.Receipt))
	if size < 1 || size > payment.ReceiptLength {
		return fault.ErrReceiptTooLong
	}

	log.Infof("pay for pay id: %v", arguments.PayId)
	//log.Infof("currency: %q", arguments.Currency)
	log.Infof("receipt: %q", arguments.Receipt)

	// announce pay block to other peers
	packed := make([]byte, len(arguments.PayId))
	copy(packed, arguments.PayId[:])
	packed = append(packed, arguments.Receipt...)

	log.Infof("broadcast pay: %x", packed)
	messagebus.Bus.Broadcast.Send("pay", packed)

	reply.Status = reservoir.TrackingAccepted

	return nil
}
