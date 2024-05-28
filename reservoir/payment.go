// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// PaymentSegment - a single segment of a payment
// e.g. for an issue or transfer block owner
type PaymentSegment [currency.Count]*transactionrecord.Payment

// get payment record from a specific block given the blocks 8 byte big endian key
func getPayments(transferBlockNumber uint64, issueBlockNumber uint64, previousTransfer transactionrecord.BitmarkTransfer, blockOwnerPaymentHandle storage.Handle) []transactionrecord.PaymentAlternative {
	if blockOwnerPaymentHandle == nil {
		return []transactionrecord.PaymentAlternative{}
	}

	tKey := make([]byte, 8)
	binary.BigEndian.PutUint64(tKey, transferBlockNumber)

	iKey := make([]byte, 8)
	binary.BigEndian.PutUint64(iKey, issueBlockNumber)

	// block owner (from issue) payment
	// 0: issue block owner
	// 1: last transfer block owner (could be merged to 1 if same address)
	// 2: transfer payment (optional)
	payments := make([]transactionrecord.PaymentAlternative, currency.Count)
	for i := 0; i < currency.Count; i += 1 {
		payments[i] = make(transactionrecord.PaymentAlternative, 1, 3)
	}

	issuePayment := getPayment(iKey, blockOwnerPaymentHandle) // will never be nil
	for i, ip := range issuePayment {
		payments[i][0] = ip
	}

	// last transfer payment if there is one otherwise issuer gets double
	transferPayment := getPayment(tKey, blockOwnerPaymentHandle)
	if transferPayment == nil {
		for _, ip := range payments {
			ip[0].Amount *= 2
		}
	} else {
		// merge to issue if the same address
		// or separate transfer payment if separate
		for i, tp := range transferPayment {
			if tp.Currency != payments[i][0].Currency {
				logger.Panicf("payment.getPayments: mismatched currencies: %s and %s", tp.Currency, payments[i][0].Currency)
			}
			if tp.Address == payments[i][0].Address {
				// transfer and issuer are the same so accumulate amount
				payments[i][0].Amount += tp.Amount
			} else {
				// separate transfer payment
				payments[i] = append(payments[i], tp)
			}
		}
	}

	// optional payment record (if previous record was transfer and contains such)
	if previousTransfer != nil && previousTransfer.GetPayment() != nil {

		i := previousTransfer.GetPayment().Currency.Index() // zero based index (panics if any problem)

		// always keep this as a separate amount even if address is the same
		// so it shows up separately in currency transaction
		payments[i] = append(payments[i], previousTransfer.GetPayment())

		return []transactionrecord.PaymentAlternative{payments[i]}
	}

	return payments
}

// get a payment record from a specific block given the blocks 8 byte big endian key
func getPayment(blockNumberKey []byte, blockOwnerPaymentHandle storage.Handle) *PaymentSegment {
	if blockOwnerPaymentHandle == nil {
		return nil
	}

	if len(blockNumberKey) != 8 {
		logger.Panicf("payment.getPayment: block number need 8 bytes: %x", blockNumberKey)
	}

	// if all 8 bytes are zero then no transfer payment as this is an issue
	if bytes.Equal([]byte{0, 0, 0, 0, 0, 0, 0, 0}, blockNumberKey) {
		return nil
	}

	paymentData := blockOwnerPaymentHandle.Get(blockNumberKey)
	if paymentData == nil {
		logger.Panicf("payment.getPayment: no block payment data for block number: %x", blockNumberKey)
	}

	cMap, _, err := currency.UnpackMap(paymentData, mode.IsTesting())
	if err != nil {
		logger.Panicf("payment.getPayment: block payment data error: %s", err)
	}

	payments := &PaymentSegment{}

	for c, address := range cMap {
		i := c.Index() // zero based index
		fee, err := c.GetFee()
		if err != nil {
			logger.Panicf("payment.getPayment: get fee returned error: %s", err)
		}

		payments[i] = &transactionrecord.Payment{
			Currency: c,
			Address:  address,
			Amount:   fee,
		}
	}

	return payments
}
