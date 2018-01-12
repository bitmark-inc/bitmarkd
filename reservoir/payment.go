// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package reservoir

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/storage"
	"github.com/bitmark-inc/bitmarkd/transactionrecord"
	"github.com/bitmark-inc/logger"
)

// a single segment of a payment
// e.g. for an issue or transfer block owner
type PaymentSegment [currency.Count]*transactionrecord.Payment

// get payment record from a specific block given the blocks 8 byte big endian key
func getPayments(ownerData []byte, previousTransfer transactionrecord.BitmarkTransfer) []transactionrecord.PaymentAlternative {

	// get block number of transfer and issue; see: storage/doc.go to determine offsets
	const transferBlockNumberOffset = merkle.DigestLength
	const issueBlockNumberOffset = 8 + 2*merkle.DigestLength

	tKey := ownerData[transferBlockNumberOffset : transferBlockNumberOffset+8]
	iKey := ownerData[issueBlockNumberOffset : issueBlockNumberOffset+8]

	// block owner (from issue) payment
	// 0: issue block owner
	// 1: last transfer block owner (could be merged to 1 if same address)
	// 2: transfer payment (optional)
	payments := make([]transactionrecord.PaymentAlternative, currency.Count)
	for i := 0; i < currency.Count; i += 1 {
		payments[i] = make(transactionrecord.PaymentAlternative, 1, 3)
	}

	issuePayment := getPayment(iKey) // will never be nil
	for i, ip := range issuePayment {
		payments[i][0] = ip
	}

	// last transfer payment if there is one otherwise issuer gets double
	transferPayment := getPayment(tKey)
	if nil == transferPayment {
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
	if nil != previousTransfer && nil != previousTransfer.GetPayment() {

		i := previousTransfer.GetPayment().Currency.Index() // zero based index (panics if any problem)

		// always keep this as a separate amount even if address is the same
		// so it shows up separately in currency transaction
		payments[i] = append(payments[i], previousTransfer.GetPayment())

		return []transactionrecord.PaymentAlternative{payments[i]}
	}

	return payments
}

// get a payment record from a specific block given the blocks 8 byte big endian key
func getPayment(blockNumberKey []byte) *PaymentSegment {

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

	// trim any trailing '0x00' bytes
trim_bytes:
	for l := len(blockOwnerData) - 1; l >= 0; l -= 1 {
		if 0x00 != blockOwnerData[l] {
			break trim_bytes
		}
		blockOwnerData = blockOwnerData[:l]
	}

	// split up individual addresses - these are in currency.Index() order
	addresses := bytes.Split(blockOwnerData, []byte{0x00})
	if len(addresses) != currency.Count {
		logger.Panicf("payment.getPayment: block owner data %#v has: %d addresses must have exactly: %d addresses", blockOwnerData, len(addresses), currency.Count)
	}
	payments := &PaymentSegment{}

	for c := currency.First; c <= currency.Last; c += 1 {
		i := c.Index() // zero based index
		fee, err := c.GetFee()
		if nil != err {
			logger.Panicf("payment.getPayment: get fee returned error: %s", err)
		}

		payments[i] = &transactionrecord.Payment{
			Currency: c,
			Address:  string(addresses[i]),
			Amount:   fee,
		}
	}

	return payments
}
