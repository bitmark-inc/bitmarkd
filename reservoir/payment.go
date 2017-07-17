// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// get payment record from a specific block given the blocks 8 byte big endian key
func getPayments(ownerData []byte, previousTransfer *transactionrecord.BitmarkTransfer) []*transactionrecord.Payment {

	// get block number of transfer and issue; see: storage/doc.go to determine offsets
	const transferBlockNumberOffset = merkle.DigestLength
	const issueBlockNumberOffset = 8 + 2*merkle.DigestLength

	tKey := ownerData[transferBlockNumberOffset : transferBlockNumberOffset+8]
	iKey := ownerData[issueBlockNumberOffset : issueBlockNumberOffset+8]

	// block owner (from issue) payment
	// 0: issue block owner
	// 1: last transfer block owner (could be merged to 1 if same address)
	// 2: transfer payment (optional)
	payments := make([]*transactionrecord.Payment, 1, 3)
	payments[0] = getPayment(iKey) // should never be nil

	// last transfer payment if there is one otherwise add it issuer
	p := getPayment(tKey)
	if nil == p {
		// no transfer payment so issuer get double
		payments[0].Amount *= 2
	} else if p.Currency == payments[0].Currency && p.Address == payments[0].Address {
		// transfer and issuer are the same so accumulate amount
		payments[0].Amount += p.Amount
	} else {
		// separate transfer payment
		payments = append(payments, p)
	}

	// optional payment record (if previous record was transfer and contains such)
	if nil != previousTransfer && nil != previousTransfer.Payment {
		// always keep this as a separate amount even if address is the same
		// so it shows up separately in currency transaction
		payments = append(payments, previousTransfer.Payment)
	}

	return payments
}

// get a payment record from a specific block given the blocks 8 byte big endian key
func getPayment(blockNumberKey []byte) *transactionrecord.Payment {

	if 8 != len(blockNumberKey) {
		logger.Panicf("payment.getPayment: block number need 8 bytes: %x", blockNumberKey)
	}

	// if all 8 bytes are zero then no transfer payment as this is an issue
	if bytes.Equal([]byte{0, 0, 0, 0, 0, 0, 0, 0}, blockNumberKey) {
		return nil
	}

	blockOwnerData := storage.Pool.BlockOwners.Get(blockNumberKey)
	if nil == blockOwnerData {
		logger.Panicf("payment.getPayment: no block owner data for block number: %x", blockNumberKey)
	}

	c, err := currency.FromUint64(binary.BigEndian.Uint64(blockOwnerData[:8]))
	if nil != err {
		logger.Panicf("payment.getPayment: block currency invalid error: %v", err)
	}

	fee, err := c.GetFee()
	if nil != err {
		logger.Panicf("payment.getPayment: get fee returned error: %v", err)
	}

	return &transactionrecord.Payment{
		Currency: c,
		Address:  string(blockOwnerData[8:]),
		Amount:   fee,
	}
}
