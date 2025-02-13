// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package share

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/rpc/owner"
	"github.com/bitmark-inc/bitmarkd/rpc/ratelimit"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
	"golang.org/x/time/rate"
)

// Share
// --------

const (
	rateLimitShare = 200
	rateBurstShare = 100
)

// Share - type for RPC
type Share struct {
	Log          *logger.L
	Limiter      *rate.Limiter
	IsNormalMode func(mode.Mode) bool
	Rsvr         reservoir.Reservoir
	ReadOnly     bool
}

func New(log *logger.L,
	isNormalMode func(mode.Mode) bool,
	rsvr reservoir.Reservoir,
	readOnly bool,
) *Share {
	return &Share{
		Log:          log,
		Limiter:      rate.NewLimiter(rateLimitShare, rateBurstShare),
		IsNormalMode: isNormalMode,
		Rsvr:         rsvr,
		ReadOnly:     readOnly,
	}
}

// Create a share with initial balance
// -----------------------------------

// CreateReply - results from creating a share
type CreateReply struct {
	TxId     merkle.Digest                                   `json:"txId"`
	ShareId  merkle.Digest                                   `json:"shareId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Create - create fractional bitmark
func (share *Share) Create(bmfr *transactionrecord.BitmarkShare, reply *CreateReply) error {

	if err := ratelimit.Limit(share.Limiter); err != nil {
		return err
	}
	if share.ReadOnly {
		return fault.NotAvailableInReadOnlyMode
	}

	log := share.Log

	log.Infof("Share.Create: %+v", bmfr)

	if !share.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	// save transfer/check for duplicate
	stored, duplicate, err := share.Rsvr.StoreTransfer(bmfr)
	if err != nil {
		return err
	}

	payId := stored.Id
	txId := stored.TxId
	shareId := stored.IssueTxId
	packed := stored.Packed

	log.Debugf("id: %v", txId)
	log.Debugf("share id: %v", shareId)

	reply.TxId = txId
	reply.ShareId = shareId
	reply.PayId = payId
	reply.Payments = make(map[string]transactionrecord.PaymentAlternative)

	for _, payment := range stored.Payments {
		c := payment[0].Currency.String()
		reply.Payments[c] = payment
	}

	// announce transaction block to other peers
	if !duplicate {
		messagebus.Bus.Broadcast.Send("transfer", packed)
	}

	return nil
}

// Get share balance
// --------------------

// BalanceArguments - arguments for RPC
type BalanceArguments struct {
	Owner   *account.Account `json:"owner"` // base58
	ShareId merkle.Digest    `json:"shareId"`
	Count   int              `json:"count"` // number of records
}

// BalanceReply - balances of shares belonging to an account
type BalanceReply struct {
	Balances []reservoir.BalanceInfo `json:"balances"`
}

// Balance - list balances for an account
func (share *Share) Balance(arguments *BalanceArguments, reply *BalanceReply) error {

	if err := ratelimit.Limit(share.Limiter); err != nil {
		return err
	}

	log := share.Log

	log.Infof("Share.Balance: %+v", arguments)

	if arguments == nil || arguments.Owner == nil {
		return fault.InvalidItem
	}

	count := arguments.Count
	if count <= 0 {
		return fault.InvalidCount
	}
	if count > owner.MaximumBitmarksCount {
		count = owner.MaximumBitmarksCount
	}

	if !share.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	if arguments.Owner.IsTesting() != mode.IsTesting() {
		return fault.WrongNetworkForPublicKey
	}

	result, err := share.Rsvr.ShareBalance(arguments.Owner, arguments.ShareId, arguments.Count)
	if err != nil {
		return err
	}

	reply.Balances = result

	return nil
}

// Grant some shares
// -----------------

// GrantReply - result of granting some shares to another account
type GrantReply struct {
	Remaining uint64                                          `json:"remaining"`
	TxId      merkle.Digest                                   `json:"txId"`
	PayId     pay.PayId                                       `json:"payId"`
	Payments  map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Grant - grant a number of shares to another account
func (share *Share) Grant(arguments *transactionrecord.ShareGrant, reply *GrantReply) error {

	if err := ratelimit.Limit(share.Limiter); err != nil {
		return err
	}
	if share.ReadOnly {
		return fault.NotAvailableInReadOnlyMode
	}

	log := share.Log

	log.Infof("Share.Grant: %+v", arguments)

	if arguments == nil || arguments.Owner == nil || arguments.Recipient == nil {
		return fault.InvalidItem
	}

	if arguments.Quantity < 1 {
		return fault.ShareQuantityTooSmall
	}

	if !share.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	if arguments.Owner.IsTesting() != mode.IsTesting() {
		return fault.WrongNetworkForPublicKey
	}

	if arguments.Recipient.IsTesting() != mode.IsTesting() {
		return fault.WrongNetworkForPublicKey
	}

	// save transfer/check for duplicate
	stored, duplicate, err := share.Rsvr.StoreGrant(arguments)
	if err != nil {
		return err
	}

	// only first result needs to be considered
	payId := stored.Id
	txId := stored.TxId
	packed := stored.Packed

	log.Debugf("id: %v", txId)
	reply.Remaining = stored.Remaining
	reply.TxId = txId
	reply.PayId = payId
	reply.Payments = make(map[string]transactionrecord.PaymentAlternative)

	for _, payment := range stored.Payments {
		c := payment[0].Currency.String()
		reply.Payments[c] = payment
	}

	// announce transaction block to other peers
	if !duplicate {
		messagebus.Bus.Broadcast.Send("transfer", packed)
	}

	return nil
}

// Swap some shares
// -------------------

// SwapReply - result of a share swap
type SwapReply struct {
	RemainingOne uint64                                          `json:"remainingOne"`
	RemainingTwo uint64                                          `json:"remainingTwo"`
	TxId         merkle.Digest                                   `json:"txId"`
	PayId        pay.PayId                                       `json:"payId"`
	Payments     map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Swap - atomically swap shares between accounts
func (share *Share) Swap(arguments *transactionrecord.ShareSwap, reply *SwapReply) error {

	if err := ratelimit.Limit(share.Limiter); err != nil {
		return err
	}
	if share.ReadOnly {
		return fault.NotAvailableInReadOnlyMode
	}

	log := share.Log

	log.Infof("Share.Swap: %+v", arguments)

	if arguments == nil || arguments.OwnerOne == nil || arguments.OwnerTwo == nil {
		return fault.InvalidItem
	}

	if arguments.QuantityOne < 1 || arguments.QuantityTwo < 1 {
		return fault.ShareQuantityTooSmall
	}

	if !share.IsNormalMode(mode.Normal) {
		return fault.NotAvailableDuringSynchronise
	}

	if arguments.OwnerOne.IsTesting() != mode.IsTesting() {
		return fault.WrongNetworkForPublicKey
	}

	if arguments.OwnerTwo.IsTesting() != mode.IsTesting() {
		return fault.WrongNetworkForPublicKey
	}

	// save transfer/check for duplicate
	stored, duplicate, err := share.Rsvr.StoreSwap(arguments)
	if err != nil {
		return err
	}

	// only first result needs to be considered
	payId := stored.Id
	txId := stored.TxId
	packed := stored.Packed

	log.Debugf("id: %v", txId)
	reply.RemainingOne = stored.RemainingOne
	reply.RemainingTwo = stored.RemainingTwo
	reply.TxId = txId
	reply.PayId = payId
	reply.Payments = make(map[string]transactionrecord.PaymentAlternative)

	for _, payment := range stored.Payments {
		c := payment[0].Currency.String()
		reply.Payments[c] = payment
	}

	// announce transaction block to other peers
	if !duplicate {
		messagebus.Bus.Broadcast.Send("transfer", packed)
	}

	return nil
}
