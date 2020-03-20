// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package bitmarks

import (
	"encoding/hex"

	"github.com/bitmark-inc/bitmarkd/rpc/ratelimit"

	"github.com/bitmark-inc/bitmarkd/rpc/assets"

	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/payment"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/logger"
)

const (
	rateLimitBitmarks = 200
	rateBurstBitmarks = reservoir.MaximumIssues
)

// Bitmarks - type for the RPC
type Bitmarks struct {
	Log                   *logger.L
	Limiter               *rate.Limiter
	IsNormalMode          func(mode.Mode) bool
	Rsvr                  reservoir.Reservoir
	PoolAssets            storage.Handle
	PoolBlockOwnerPayment storage.Handle
}

// IssueStatus - results from an issue
type IssueStatus struct {
	TxId merkle.Digest `json:"txId"`
}

// CreateArguments - arguments for creating a bitmark
type CreateArguments struct {
	Assets []*transactionrecord.AssetData    `json:"assets"`
	Issues []*transactionrecord.BitmarkIssue `json:"issues"`
}

func New(log *logger.L, pools reservoir.Handles, isNormalMode func(mode.Mode) bool, rsvr reservoir.Reservoir) *Bitmarks {
	return &Bitmarks{
		Log:                   log,
		Limiter:               rate.NewLimiter(rateLimitBitmarks, rateBurstBitmarks),
		IsNormalMode:          isNormalMode,
		Rsvr:                  rsvr,
		PoolAssets:            pools.Assets,
		PoolBlockOwnerPayment: pools.BlockOwnerPayment,
	}
}

// CreateReply - results from create RPC
type CreateReply struct {
	Assets     []assets.Status                                 `json:"assets"`
	Issues     []IssueStatus                                   `json:"issues"`
	PayId      pay.PayId                                       `json:"payId"`
	PayNonce   reservoir.PayNonce                              `json:"payNonce"`
	Difficulty string                                          `json:"difficulty,omitempty"`
	Payments   map[string]transactionrecord.PaymentAlternative `json:"payments,omitempty"`
}

// Create - create assets and issues
func (bitmarks *Bitmarks) Create(arguments *CreateArguments, reply *CreateReply) error {

	log := bitmarks.Log
	assetCount := len(arguments.Assets)
	issueCount := len(arguments.Issues)

	if assetCount > reservoir.MaximumIssues || issueCount > reservoir.MaximumIssues {
		return fault.TooManyItemsToProcess
	} else if 0 == assetCount && 0 == issueCount {
		return fault.MissingParameters
	}

	count := assetCount + issueCount
	if count > reservoir.MaximumIssues {
		count = reservoir.MaximumIssues
	}
	if err := ratelimit.LimitN(bitmarks.Limiter, count, reservoir.MaximumIssues); nil != err {
		return err
	}

	if !bitmarks.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	log.Infof("Bitmarks.Create: %+v", arguments)

	assetStatus, packedAssets, err := assets.Register(arguments.Assets, bitmarks.PoolAssets)
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
		stored, duplicate, err = bitmarks.Rsvr.StoreIssues(arguments.Issues)
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
		return fault.MissingParameters
	}
	// if data to send
	if 0 != len(packedAssets) {
		// announce transaction block to other peers
		messagebus.Bus.Broadcast.Send("assets", packedAssets)
	}

	if 0 != len(packedIssues) {

		result.PayId = stored.Id
		result.PayNonce = stored.Nonce
		if nil == stored.Difficulty {
			result.Difficulty = "" // suppress difficulty if not applicable
		} else {
			result.Difficulty = stored.Difficulty.GoString()
		}
		if nil != stored.Payments {
			result.Payments = make(map[string]transactionrecord.PaymentAlternative)

			for _, p := range stored.Payments {
				c := p[0].Currency.String()
				result.Payments[c] = p
			}
		}

		// announce transaction block to other peers
		if !duplicate {
			messagebus.Bus.Broadcast.Send("issues", packedIssues, util.ToVarint64(0))
		}
	}

	log.Infof("Bitmarks.Create: result: %#v", result)
	*reply = result
	return nil
}

// Bitmarks proof
// --------------

// ProofArguments - arguments for RPC
type ProofArguments struct {
	PayId pay.PayId `json:"payId"`
	Nonce string    `json:"nonce"`
}

// ProofReply - results from a proof RPC
type ProofReply struct {
	Status reservoir.TrackingStatus `json:"status"`
}

// Proof - supply proof that client-side hashing to confirm free issue was done
func (bitmarks *Bitmarks) Proof(arguments *ProofArguments, reply *ProofReply) error {
	if err := ratelimit.Limit(bitmarks.Limiter); nil != err {
		return err
	}

	log := bitmarks.Log

	if !bitmarks.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	// arbitrary byte size limit
	size := hex.DecodedLen(len(arguments.Nonce))
	if size < payment.MinimumNonceLength || size > payment.MaximumNonceLength {
		return fault.InvalidNonce
	}

	log.Infof("proof for pay id: %v", arguments.PayId)
	log.Infof("client nonce: %q", arguments.Nonce)

	nonce := make([]byte, size)
	byteCount, err := hex.Decode(nonce, []byte(arguments.Nonce))
	if nil != err {
		return err
	}
	if byteCount != size {
		return fault.InvalidNonce
	}

	log.Infof("client nonce hex: %x", nonce)

	// announce proof block to other peers
	packed := make([]byte, len(arguments.PayId), len(arguments.PayId)+len(nonce))
	copy(packed, arguments.PayId[:])
	packed = append(packed, nonce...)

	log.Infof("broadcast proof: %x", packed)
	messagebus.Bus.Broadcast.Send("proof", packed)

	// check if proof matches
	reply.Status = bitmarks.Rsvr.TryProof(arguments.PayId, nonce)

	return nil
}
