// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package payment

import (
	"bytes"
	"encoding/binary"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
)

// get payment record from a specific block given the blocks 8 byte big endian key
func GetPayments(ownerData []byte, previousTransfer *transactionrecord.BitmarkTransfer) []*transactionrecord.Payment {

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
	payments[0] = GetPayment(iKey) // should never be nil

	// last transfer payment if there is one otherwise add it issuer
	p := GetPayment(tKey)
	if nil == p || (p.Currency == payments[0].Currency && p.Address == payments[0].Address) {
		payments[0].Amount += p.Amount
	} else {
		payments = append(payments, p)
	}

	// optional payment record (if previous record was transfer and contains such)
	if nil != previousTransfer && nil != previousTransfer.Payment {
		payments = append(payments, previousTransfer.Payment)
	}

	return payments
}

// get a payment record from a specific block given the blocks 8 byte big endian key
func GetPayment(blockNumberKey []byte) *transactionrecord.Payment {

	if 8 != len(blockNumberKey) {
		fault.Panicf("payment.GetPayment: block number need 8 bytes: %x", blockNumberKey)
	}
	if bytes.Equal([]byte{0, 0, 0, 0, 0, 0, 0}, blockNumberKey) {
		return nil
	}

	blockOwnerData := storage.Pool.BlockOwners.Get(blockNumberKey)
	if nil == blockOwnerData {
		fault.Panicf("payment.GetPayment: no block owner data for block number: %x", blockNumberKey)
	}

	c, err := currency.FromUint64(binary.BigEndian.Uint64(blockOwnerData[:8]))
	if nil != err {
		fault.Panicf("payment.GetPayment: block currency invalid error: %v", err)
	}

	fee, err := c.GetFee()
	if nil != err {
		fault.Panicf("payment.GetPayment: get fee returned error: %v", err)
	}

	return &transactionrecord.Payment{
		Currency: c,
		Address:  string(blockOwnerData[8:]),
		Amount:   fee,
	}
}
