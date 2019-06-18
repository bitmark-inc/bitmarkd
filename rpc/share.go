// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"golang.org/x/time/rate"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/messagebus"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/pay"
	"github.com/bitmark-inc/bitmarkd/reservoir"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// Share
// --------

// Share - type for RPC
type Share struct {
	log     *logger.L
	limiter *rate.Limiter
}

// Create a share with initial balance
// -----------------------------------

// ShareCreateReply - results from creating a share
type ShareCreateReply struct {
	TxId     merkle.Digest                                   `json:"txId"`
	ShareId  merkle.Digest                                   `json:"shareId"`
	PayId    pay.PayId                                       `json:"payId"`
	Payments map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Create - create fractional bitmark
func (share *Share) Create(bmfr *transactionrecord.BitmarkShare, reply *ShareCreateReply) error {

	if err := rateLimit(share.limiter); nil != err {
		return err
	}

	log := share.log

	log.Infof("Share.Create: %+v", bmfr)

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	// save transfer/check for duplicate
	stored, duplicate, err := reservoir.StoreTransfer(bmfr)
	if nil != err {
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

// ShareBalanceArguments - arguments for RPC
type ShareBalanceArguments struct {
	Owner   *account.Account `json:"owner"` // base58
	ShareId merkle.Digest    `json:"shareId"`
	Count   int              `json:"count"` // number of records
}

// ShareBalanceReply - balances of shares belonging to an account
type ShareBalanceReply struct {
	Balances []reservoir.BalanceInfo `json:"balances"`
}

// Balance - list balances for an account
func (share *Share) Balance(arguments *ShareBalanceArguments, reply *ShareBalanceReply) error {
	var count int

	if err := rateLimit(share.limiter); nil != err {
		return err
	}

	log := share.log

	log.Infof("Share.Balance: %+v", arguments)

	if nil == arguments || nil == arguments.Owner {
		return fault.ErrInvalidItem
	}

	count = arguments.Count
	if count <= 0 {
		return fault.ErrInvalidCount
	}
	if count > maximumBitmarksCount {
		count = maximumBitmarksCount
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	if arguments.Owner.IsTesting() != mode.IsTesting() {
		return fault.ErrWrongNetworkForPublicKey
	}

	result, err := reservoir.ShareBalance(arguments.Owner, arguments.ShareId, arguments.Count)
	if nil != err {
		return err
	}

	reply.Balances = result

	return nil
}

// Grant some shares
// -----------------

// ShareGrantReply - result of granting some shares to another account
type ShareGrantReply struct {
	Remaining uint64                                          `json:"remaining"`
	TxId      merkle.Digest                                   `json:"txId"`
	PayId     pay.PayId                                       `json:"payId"`
	Payments  map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Grant - grant a number of shares to another account
func (share *Share) Grant(arguments *transactionrecord.ShareGrant, reply *ShareGrantReply) error {

	if err := rateLimit(share.limiter); nil != err {
		return err
	}

	log := share.log

	log.Infof("Share.Grant: %+v", arguments)

	if nil == arguments || nil == arguments.Owner || nil == arguments.Recipient {
		return fault.ErrInvalidItem
	}

	if arguments.Quantity < 1 {
		return fault.ErrShareQuantityTooSmall
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	if arguments.Owner.IsTesting() != mode.IsTesting() {
		return fault.ErrWrongNetworkForPublicKey
	}

	if arguments.Recipient.IsTesting() != mode.IsTesting() {
		return fault.ErrWrongNetworkForPublicKey
	}

	// save transfer/check for duplicate
	stored, duplicate, err := reservoir.StoreGrant(arguments)
	if nil != err {
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

// ShareSwapReply - result of a share swap
type ShareSwapReply struct {
	RemainingOne uint64                                          `json:"remainingOne"`
	RemainingTwo uint64                                          `json:"remainingTwo"`
	TxId         merkle.Digest                                   `json:"txId"`
	PayId        pay.PayId                                       `json:"payId"`
	Payments     map[string]transactionrecord.PaymentAlternative `json:"payments"`
}

// Swap - atomically swap shares between accounts
func (share *Share) Swap(arguments *transactionrecord.ShareSwap, reply *ShareSwapReply) error {

	if err := rateLimit(share.limiter); nil != err {
		return err
	}

	log := share.log

	log.Infof("Share.Swap: %+v", arguments)

	if nil == arguments || nil == arguments.OwnerOne || nil == arguments.OwnerTwo {
		return fault.ErrInvalidItem
	}

	if arguments.QuantityOne < 1 || arguments.QuantityTwo < 1 {
		return fault.ErrShareQuantityTooSmall
	}

	if !mode.Is(mode.Normal) {
		return fault.ErrNotAvailableDuringSynchronise
	}

	if arguments.OwnerOne.IsTesting() != mode.IsTesting() {
		return fault.ErrWrongNetworkForPublicKey
	}

	if arguments.OwnerTwo.IsTesting() != mode.IsTesting() {
		return fault.ErrWrongNetworkForPublicKey
	}

	// save transfer/check for duplicate
	stored, duplicate, err := reservoir.StoreSwap(arguments)
	if nil != err {
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
