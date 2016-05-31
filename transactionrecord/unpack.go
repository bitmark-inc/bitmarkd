// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/util"
)

// turn a byte slice into a record
//
// must cast result to correct type
//
// e.g.
//   registration, ok := result.(*transaction.Registration)
// or:
//   switch result.(type) {
//   case *transaction.Registration:
func (record Packed) Unpack() (interface{}, error) {

	recordType, n := util.FromVarint64(record)

	switch TagType(recordType) {

	case BaseDataTag:

		// currency
		c := uint64(0)
		var currencyLength int
		c, currencyLength = util.FromVarint64(record[n:])
		n += int(currencyLength)
		currency, err := currency.FromUint64(c)
		if nil != err {
			return nil, err
		}

		// paymentAddress
		paymentAddressLength, paymentAddressOffset := util.FromVarint64(record[n:])
		paymentAddress := make([]byte, paymentAddressLength)
		n += paymentAddressOffset
		copy(paymentAddress, record[n:])
		n += int(paymentAddressLength)

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, err
		}
		n += int(ownerLength)

		// nonce
		nonce := uint64(0)
		var nonceLength int
		nonce, nonceLength = util.FromVarint64(record[n:])
		n += int(nonceLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &BaseData{
			Owner:          owner,
			Currency:       currency,
			PaymentAddress: string(paymentAddress),
			Nonce:          nonce,
			Signature:      signature,
		}
		return r, nil

	case AssetDataTag:

		// description
		descriptionLength, descriptionOffset := util.FromVarint64(record[n:])
		description := make([]byte, descriptionLength)
		n += descriptionOffset
		copy(description, record[n:])
		n += int(descriptionLength)

		// name
		nameLength, nameOffset := util.FromVarint64(record[n:])
		name := make([]byte, nameLength)
		n += nameOffset
		copy(name, record[n:])
		n += int(nameLength)

		// fingerprint
		fingerprintLength, fingerprintOffset := util.FromVarint64(record[n:])
		fingerprint := make([]byte, fingerprintLength)
		n += fingerprintOffset
		copy(fingerprint, record[n:])
		n += int(fingerprintLength)

		// registrant public key
		registrantLength, registrantOffset := util.FromVarint64(record[n:])
		n += registrantOffset
		registrant, err := account.AccountFromBytes(record[n : n+int(registrantLength)])
		if nil != err {
			return nil, err
		}
		n += int(registrantLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &AssetData{
			Description: string(description),
			Name:        string(name),
			Fingerprint: string(fingerprint),
			Registrant:  registrant,
			Signature:   signature,
		}
		return r, nil

	case BitmarkIssueTag:

		// asset index
		assetIndexLength, assetIndexOffset := util.FromVarint64(record[n:])
		n += assetIndexOffset
		var assetIndex AssetIndex
		err := AssetIndexFromBytes(&assetIndex, record[n:n+int(assetIndexLength)])
		if nil != err {
			return nil, err
		}
		n += int(assetIndexLength)

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, err
		}
		n += int(ownerLength)

		// nonce
		nonce := uint64(0)
		var nonceLength int
		nonce, nonceLength = util.FromVarint64(record[n:])
		n += int(nonceLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &BitmarkIssue{
			AssetIndex: assetIndex,
			Owner:      owner,
			Signature:  signature,
			Nonce:      nonce,
		}
		return r, nil

	case BitmarkTransferTag:

		// link
		linkLength, linkOffset := util.FromVarint64(record[n:])
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+int(linkLength)])
		if nil != err {
			return nil, err
		}
		n += int(linkLength)

		// optional payment
		payment := (*Payment)(nil)

		if 0 == record[n] {
			n += 1
		} else if 1 == record[n] {
			n += 1

			// currency
			c := uint64(0)
			var currencyLength int
			c, currencyLength = util.FromVarint64(record[n:])
			n += int(currencyLength)
			currency, err := currency.FromUint64(c)
			if nil != err {
				return nil, err
			}

			// address
			addressLength, addressOffset := util.FromVarint64(record[n:])
			address := make([]byte, addressLength)
			n += addressOffset
			copy(address, record[n:])
			n += int(addressLength)

			// amount
			amount, amountLength := util.FromVarint64(record[n:])
			n += int(amountLength)

			payment = &Payment{
				Currency: currency,
				Address:  string(address),
				Amount:   amount,
			}
		} else {
			return nil, fault.ErrNotTransactionPack
		}

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, err
		}
		n += int(ownerLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &BitmarkTransfer{
			Link:      Link(link),
			Payment:   payment,
			Owner:     owner,
			Signature: signature,
		}
		return r, nil

	default:
	}
	return nil, fault.ErrNotTransactionPack
}
