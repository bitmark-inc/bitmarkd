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
// Note: the unpacker will access the underlying array of the packed
//       record so p[x:y].Unpack() can read past p[y] and couldcontinue up to cap(p)
//       i.e p[x:cap(p)].Unpack() performs the same operation
//       elements beefore p[x] cannot be accessed
//       see: https://blog.golang.org/go-slices-usage-and-internals
//
// must cast result to correct type
//
// e.g.
//   registration, ok := result.(*transaction.Registration)
// or:
//   switch tx := result.(type) {
//   case *transaction.Registration:
func (record Packed) Unpack(testnet bool) (t Transaction, n int, e error) {

	defer func() {
		if r := recover(); nil != r {
			e = fault.ErrNotTransactionPack
		}
	}()

	recordType, n := util.ClippedVarint64(record, 1, 8192)
	if 0 == n {
		return nil, 0, fault.ErrNotTransactionPack
	}

unpack_switch:
	switch TagType(recordType) {

	case BaseDataTag:

		// currency
		c, currencyLength := util.FromVarint64(record[n:])
		if 0 == currencyLength {
			break unpack_switch
		}
		n += currencyLength
		currency, err := currency.FromUint64(c)
		if nil != err {
			return nil, 0, err
		}

		// paymentAddress
		paymentAddressLength, paymentAddressOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == paymentAddressOffset {
			break unpack_switch
		}
		n += paymentAddressOffset
		paymentAddress := string(record[n : n+paymentAddressLength])
		n += paymentAddressLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == ownerOffset {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += ownerLength

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		if 0 == nonceLength {
			break unpack_switch
		}
		n += nonceLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == signatureOffset {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

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
		nameLength, nameOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == nameOffset {
			break unpack_switch
		}
		name := make([]byte, nameLength)
		n += nameOffset
		copy(name, record[n:n+nameLength])
		n += nameLength

		// fingerprint
		fingerprintLength, fingerprintOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == fingerprintOffset {
			break unpack_switch
		}
		fingerprint := make([]byte, fingerprintLength)
		n += fingerprintOffset
		copy(fingerprint, record[n:n+fingerprintLength])
		n += fingerprintLength

		// metadata (can be zero length)
		metadataLength, metadataOffset := util.ClippedVarint64(record[n:], 0, 8192) // Note: zero is valid here
		if 0 == metadataOffset {
			break unpack_switch
		}
		metadata := make([]byte, metadataLength)
		n += metadataOffset
		copy(metadata, record[n:n+metadataLength])
		n += metadataLength

		// registrant public key
		registrantLength, registrantOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == registrantOffset {
			break unpack_switch
		}
		n += registrantOffset
		registrant, err := account.AccountFromBytes(record[n : n+registrantLength])
		if nil != err {
			return nil, 0, err
		}
		if registrant.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += registrantLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == signatureOffset {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		r := &AssetData{
			Name:        string(name),
			Fingerprint: string(fingerprint),
			Metadata:    string(metadata),
			Registrant:  registrant,
			Signature:   signature,
		}
		return r, n, nil

	case BitmarkIssueTag:

		// asset id
		assetIdentifierLength, assetIdentifierOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == assetIdentifierOffset {
			break unpack_switch
		}
		n += assetIdentifierOffset
		var assetId AssetIdentifier
		err := AssetIdentifierFromBytes(&assetId, record[n:n+assetIdentifierLength])
		if nil != err {
			return nil, 0, err
		}
		n += assetIdentifierLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == ownerOffset {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += ownerLength

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		if 0 == nonceLength {
			break unpack_switch
		}
		n += nonceLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == signatureOffset {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		r := &BitmarkIssue{
			AssetId:   assetId,
			Owner:     owner,
			Signature: signature,
			Nonce:     nonce,
		}
		return r, n, nil

	case BitmarkTransferUnratifiedTag:

		// link
		linkLength, linkOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == linkOffset {
			break unpack_switch
		}
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+linkLength])
		if nil != err {
			return nil, 0, err
		}
		n += linkLength

		// optional escrow payment
		escrow, n, err := unpackEscrow(record, n)
		if nil != err {
			return nil, 0, err
		}

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == ownerOffset {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += ownerLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == signatureOffset {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		r := &BitmarkTransferUnratified{
			Link:      link,
			Escrow:    escrow,
			Owner:     owner,
			Signature: signature,
		}
		return r, n, nil

	case BitmarkTransferCountersignedTag:

		// link
		linkLength, linkOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == linkOffset {
			break unpack_switch
		}
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+linkLength])
		if nil != err {
			return nil, 0, err
		}
		n += linkLength

		// optional escrow payment
		escrow, n, err := unpackEscrow(record, n)
		if nil != err {
			return nil, 0, err
		}

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == ownerOffset {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += ownerLength

		// signature
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == signatureOffset {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		// countersignature
		countersignatureLength, countersignatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == countersignatureOffset {
			break unpack_switch
		}
		countersignature := make(account.Signature, countersignatureLength)
		n += countersignatureOffset
		copy(countersignature, record[n:n+countersignatureLength])
		n += countersignatureLength

		r := &BitmarkTransferCountersigned{
			Link:             link,
			Escrow:           escrow,
			Owner:            owner,
			Signature:        signature,
			Countersignature: countersignature,
		}
		return r, n, nil

	case BlockFoundationTag:

		// version
		version, versionLength := util.FromVarint64(record[n:])
		if 0 == versionLength {
			break unpack_switch
		}
		n += versionLength
		if version < 1 || version >= uint64(len(versions)) {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}

		// payment map
		paymentsLength, paymentsOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == paymentsOffset {
			break unpack_switch
		}
		n += paymentsOffset
		payments, cs, err := currency.UnpackMap(record[n:n+paymentsLength], testnet)
		if nil != err {
			return nil, 0, err
		}
		if cs != versions[version] {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}
		n += paymentsLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == ownerOffset {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += ownerLength

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		if 0 == nonceLength {
			break unpack_switch
		}
		n += nonceLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == signatureOffset {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		r := &BlockFoundation{
			Version:   version,
			Owner:     owner,
			Payments:  payments,
			Nonce:     nonce,
			Signature: signature,
		}
		return r, n, nil

	case BlockOwnerTransferTag:

		// link
		linkLength, linkOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == linkOffset {
			break unpack_switch
		}
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+linkLength])
		if nil != err {
			return nil, 0, err
		}
		n += linkLength

		// optional escrow payment
		escrow, n, err := unpackEscrow(record, n)
		if nil != err {
			return nil, 0, err
		}

		// version
		version, versionLength := util.FromVarint64(record[n:])
		if 0 == versionLength {
			break unpack_switch
		}
		n += versionLength
		if version < 1 || version >= uint64(len(versions)) {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}

		// payment map

		paymentsLength, paymentsOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == paymentsOffset {
			break unpack_switch
		}
		n += paymentsOffset
		payments, cs, err := currency.UnpackMap(record[n:n+paymentsLength], testnet)
		if nil != err {
			return nil, 0, err
		}
		if cs != versions[version] {
			return nil, 0, fault.ErrInvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}
		n += paymentsLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == ownerOffset {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if nil != err {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.ErrWrongNetworkForPublicKey
		}
		n += ownerLength

		// signature
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == signatureOffset {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		// countersignature
		countersignatureLength, countersignatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == countersignatureOffset {
			break unpack_switch
		}
		countersignature := make(account.Signature, countersignatureLength)
		n += countersignatureOffset
		copy(countersignature, record[n:n+countersignatureLength])
		n += countersignatureLength

		r := &BlockOwnerTransfer{
			Link:             link,
			Escrow:           escrow,
			Version:          version,
			Owner:            owner,
			Payments:         payments,
			Signature:        signature,
			Countersignature: countersignature,
		}
		return r, n, nil

	default: // also NullTag
	}
	return nil, 0, fault.ErrNotTransactionPack
}

func unpackEscrow(record []byte, n int) (*Payment, int, error) {

	// optional escrow payment
	payment := (*Payment)(nil)

	if 0 == record[n] {
		n += 1
	} else if 1 == record[n] {
		n += 1

		// currency
		c, currencyLength := util.FromVarint64(record[n:])
		if 0 == currencyLength {
			return nil, 0, fault.ErrNotTransactionPack
		}
		n += currencyLength
		currency, err := currency.FromUint64(c)
		if nil != err {
			return nil, 0, err
		}

		// address
		addressLength, addressOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if 0 == addressOffset {
			return nil, 0, fault.ErrNotTransactionPack
		}
		n += addressOffset
		address := string(record[n : n+addressLength])
		n += addressLength

		// amount
		amount, amountLength := util.FromVarint64(record[n:])
		if 0 == amountLength {
			return nil, 0, fault.ErrNotTransactionPack
		}
		n += amountLength

		payment = &Payment{
			Currency: currency,
			Address:  address,
			Amount:   amount,
		}
	} else {
		return nil, 0, fault.ErrNotTransactionPack
	}
	return payment, n, nil
}
