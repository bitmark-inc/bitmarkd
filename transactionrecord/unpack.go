// Copyright (c) 2014-2018 Bitmark Inc.
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
//   switch tx := result.(type) {
//   case *transaction.Registration:
func (record Packed) Unpack(testnet bool) (Transaction, int, error) {

	recordType, n := util.FromVarint64(record)

	switch TagType(recordType) {

	case BaseDataTag:

		// currency
		c, currencyLength := util.FromVarint64(record[n:])
		n += int(currencyLength)
		currency, err := currency.FromUint64(c)
		if nil != err {
			return nil, 0, err
		}

		// paymentAddress
		paymentAddressLength, paymentAddressOffset := util.FromVarint64(record[n:])
		n += paymentAddressOffset
		paymentAddress := string(record[n : n+int(paymentAddressLength)])
		n += int(paymentAddressLength)

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += int(ownerLength)

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		n += int(nonceLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &OldBaseData{
			Owner:          owner,
			Currency:       currency,
			PaymentAddress: string(paymentAddress),
			Nonce:          nonce,
			Signature:      signature,
		}
		return r, n, nil

	case AssetDataTag:

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

		// metadata
		metadataLength, metadataOffset := util.FromVarint64(record[n:])
		metadata := make([]byte, metadataLength)
		n += metadataOffset
		copy(metadata, record[n:])
		n += int(metadataLength)

		// registrant public key
		registrantLength, registrantOffset := util.FromVarint64(record[n:])
		n += registrantOffset
		registrant, err := account.AccountFromBytes(record[n : n+int(registrantLength)])
		if nil != err {
			return nil, 0, err
		}
		if registrant.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += int(registrantLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &AssetData{
			Name:        string(name),
			Fingerprint: string(fingerprint),
			Metadata:    string(metadata),
			Registrant:  registrant,
			Signature:   signature,
		}
		return r, n, nil

	case BitmarkIssueTag:

		// asset index
		assetIndexLength, assetIndexOffset := util.FromVarint64(record[n:])
		n += assetIndexOffset
		var assetIndex AssetIndex
		err := AssetIndexFromBytes(&assetIndex, record[n:n+int(assetIndexLength)])
		if nil != err {
			return nil, 0, err
		}
		n += int(assetIndexLength)

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
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
		return r, n, nil

	case BitmarkTransferUnratifiedTag:

		// link
		linkLength, linkOffset := util.FromVarint64(record[n:])
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+int(linkLength)])
		if nil != err {
			return nil, 0, err
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
				return nil, 0, err
			}

			// address
			addressLength, addressOffset := util.FromVarint64(record[n:])
			n += addressOffset
			address := string(record[n : n+int(addressLength)])
			n += int(addressLength)

			// amount
			amount, amountLength := util.FromVarint64(record[n:])
			n += int(amountLength)

			payment = &Payment{
				Currency: currency,
				Address:  address,
				Amount:   amount,
			}
		} else {
			return nil, 0, fault.ErrNotTransactionPack
		}

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += int(ownerLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &BitmarkTransferUnratified{
			Link:      link,
			Payment:   payment,
			Owner:     owner,
			Signature: signature,
		}
		return r, n, nil

	case BitmarkTransferCountersignedTag:

		// link
		linkLength, linkOffset := util.FromVarint64(record[n:])
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+int(linkLength)])
		if nil != err {
			return nil, 0, err
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
				return nil, 0, err
			}

			// address
			addressLength, addressOffset := util.FromVarint64(record[n:])
			n += addressOffset
			address := string(record[n : n+int(addressLength)])
			n += int(addressLength)

			// amount
			amount, amountLength := util.FromVarint64(record[n:])
			n += int(amountLength)

			payment = &Payment{
				Currency: currency,
				Address:  address,
				Amount:   amount,
			}
		} else {
			return nil, 0, fault.ErrNotTransactionPack
		}

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += int(ownerLength)

		// signature
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		// countersignature
		countersignatureLength, countersignatureOffset := util.FromVarint64(record[n:])
		countersignature := make(account.Signature, countersignatureLength)
		n += countersignatureOffset
		copy(countersignature, record[n:])
		n += int(countersignatureLength)

		r := &BitmarkTransferCountersigned{
			Link:             link,
			Payment:          payment,
			Owner:            owner,
			Signature:        signature,
			Countersignature: countersignature,
		}
		return r, n, nil

	case BlockOwnerIssueTag:

		// version
		version, versionLength := util.FromVarint64(record[n:])
		n += int(versionLength)
		if version < 1 || version >= uint64(len(versions)) {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}

		// payment map
		paymentsLength, paymentsOffset := util.FromVarint64(record[n:])
		n += paymentsOffset
		payments, cs, err := currency.UnpackMap(record[n:n+int(paymentsLength)], testnet)
		if nil != err {
			return nil, 0, err
		}
		if cs != versions[version] {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}
		n += int(paymentsLength)

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += int(ownerLength)

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		n += int(nonceLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &BlockOwnerIssue{
			Version:   version,
			Owner:     owner,
			Payments:  payments,
			Nonce:     nonce,
			Signature: signature,
		}
		return r, n, nil

	case BlockOwnerTransferTag:

		// link
		linkLength, linkOffset := util.FromVarint64(record[n:])
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+int(linkLength)])
		if nil != err {
			return nil, 0, err
		}
		n += int(linkLength)

		// version
		version, versionLength := util.FromVarint64(record[n:])
		n += int(versionLength)
		if version < 1 || version >= uint64(len(versions)) {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}

		// payment map

		paymentsLength, paymentsOffset := util.FromVarint64(record[n:])
		n += paymentsOffset
		payments, cs, err := currency.UnpackMap(record[n:n+int(paymentsLength)], testnet)
		if nil != err {
			return nil, 0, err
		}
		if cs != versions[version] {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}
		n += int(paymentsLength)

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += int(ownerLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &BlockOwnerTransfer{
			Link:      link,
			Version:   version,
			Owner:     owner,
			Payments:  payments,
			Signature: signature,
		}
		return r, n, nil

	default:
	}
	return nil, 0, fault.ErrNotTransactionPack
}
