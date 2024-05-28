// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
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

// Unpack - turn a byte slice into a record
// Note: the unpacker will access the underlying array of the packed
//
//	record so p[x:y].Unpack() can read past p[y] and couldcontinue up to cap(p)
//	i.e p[x:cap(p)].Unpack() performs the same operation
//	elements beefore p[x] cannot be accessed
//	see: https://blog.golang.org/go-slices-usage-and-internals
//
// must cast result to correct type
//
// e.g.
//
//	registration, ok := result.(*transaction.Registration)
//
// or:
//
//	switch tx := result.(type) {
//	case *transaction.Registration:
func (record Packed) Unpack(testnet bool) (t Transaction, n int, e error) {

	defer func() {
		if r := recover(); r != nil {
			e = fault.NotTransactionPack
		}
	}()

	recordType, n := util.ClippedVarint64(record, 1, 8192)
	if n == 0 {
		return nil, 0, fault.NotTransactionPack
	}

unpack_switch:
	switch TagType(recordType) {

	case BaseDataTag:

		// currency
		c, currencyLength := util.FromVarint64(record[n:])
		if currencyLength == 0 {
			break unpack_switch
		}
		n += currencyLength
		currencyValue, err := currency.FromUint64(c)
		if err != nil {
			return nil, 0, err
		}

		// paymentAddress
		paymentAddressLength, paymentAddressOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if paymentAddressOffset == 0 {
			break unpack_switch
		}
		n += paymentAddressOffset
		paymentAddress := string(record[n : n+paymentAddressLength])
		n += paymentAddressLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOffset == 0 {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if err != nil {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerLength

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		if nonceLength == 0 {
			break unpack_switch
		}
		n += nonceLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		r := &OldBaseData{
			Owner:          owner,
			Currency:       currencyValue,
			PaymentAddress: string(paymentAddress),
			Nonce:          nonce,
			Signature:      signature,
		}
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case AssetDataTag:

		// name
		nameLength, nameOffset := util.ClippedVarint64(record[n:], 0, 8192)

		name := make([]byte, nameLength)
		n += nameOffset
		copy(name, record[n:n+nameLength])
		n += nameLength

		// fingerprint
		fingerprintLength, fingerprintOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if fingerprintOffset == 0 {
			break unpack_switch
		}
		fingerprint := make([]byte, fingerprintLength)
		n += fingerprintOffset
		copy(fingerprint, record[n:n+fingerprintLength])
		n += fingerprintLength

		// metadata (can be zero length)
		metadataLength, metadataOffset := util.ClippedVarint64(record[n:], 0, 8192) // Note: zero is valid here
		if metadataOffset == 0 {
			break unpack_switch
		}
		metadata := make([]byte, metadataLength)
		n += metadataOffset
		copy(metadata, record[n:n+metadataLength])
		n += metadataLength

		// registrant public key
		registrantLength, registrantOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if registrantOffset == 0 {
			break unpack_switch
		}
		n += registrantOffset
		registrant, err := account.AccountFromBytes(record[n : n+registrantLength])
		if err != nil {
			return nil, 0, err
		}
		if registrant.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += registrantLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
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
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case BitmarkIssueTag:

		// asset id
		assetIdentifierLength, assetIdentifierOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if assetIdentifierOffset == 0 {
			break unpack_switch
		}
		n += assetIdentifierOffset
		var assetId AssetIdentifier
		err := AssetIdentifierFromBytes(&assetId, record[n:n+assetIdentifierLength])
		if err != nil {
			return nil, 0, err
		}
		n += assetIdentifierLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOffset == 0 {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if err != nil {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerLength

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		if nonceLength == 0 {
			break unpack_switch
		}
		n += nonceLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
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
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case BitmarkTransferUnratifiedTag:

		// link
		linkLength, linkOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if linkOffset == 0 {
			break unpack_switch
		}
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+linkLength])
		if err != nil {
			return nil, 0, err
		}
		n += linkLength

		// optional escrow payment
		escrow, n, err := unpackEscrow(record, n)
		if err != nil {
			return nil, 0, err
		}

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOffset == 0 {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if err != nil {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
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
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case BitmarkTransferCountersignedTag:

		// link
		linkLength, linkOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if linkOffset == 0 {
			break unpack_switch
		}
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+linkLength])
		if err != nil {
			return nil, 0, err
		}
		n += linkLength

		// optional escrow payment
		escrow, n, err := unpackEscrow(record, n)
		if err != nil {
			return nil, 0, err
		}

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOffset == 0 {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if err != nil {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerLength

		// signature
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		// countersignature
		countersignatureLength, countersignatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if countersignatureOffset == 0 {
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
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case BlockFoundationTag:

		// version
		version, versionLength := util.FromVarint64(record[n:])
		if versionLength == 0 {
			break unpack_switch
		}
		n += versionLength
		if version < 1 || version >= uint64(len(versions)) {
			return nil, 0, fault.InvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}

		// payment map
		paymentsLength, paymentsOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if paymentsOffset == 0 {
			break unpack_switch
		}
		n += paymentsOffset
		payments, cs, err := currency.UnpackMap(record[n:n+paymentsLength], testnet)
		if err != nil {
			return nil, 0, err
		}
		if cs != versions[version] {
			return nil, 0, fault.InvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}
		n += paymentsLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOffset == 0 {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if err != nil {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerLength

		// nonce
		nonce, nonceLength := util.FromVarint64(record[n:])
		if nonceLength == 0 {
			break unpack_switch
		}
		n += nonceLength

		// signature is remainder of record
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
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
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case BlockOwnerTransferTag:

		// link
		linkLength, linkOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if linkOffset == 0 {
			break unpack_switch
		}
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+linkLength])
		if err != nil {
			return nil, 0, err
		}
		n += linkLength

		// optional escrow payment
		escrow, n, err := unpackEscrow(record, n)
		if err != nil {
			return nil, 0, err
		}

		// version
		version, versionLength := util.FromVarint64(record[n:])
		if versionLength == 0 {
			break unpack_switch
		}
		n += versionLength
		if version < 1 || version >= uint64(len(versions)) {
			return nil, 0, fault.InvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}

		// payment map

		paymentsLength, paymentsOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if paymentsOffset == 0 {
			break unpack_switch
		}
		n += paymentsOffset
		payments, cs, err := currency.UnpackMap(record[n:n+paymentsLength], testnet)
		if err != nil {
			return nil, 0, err
		}
		if cs != versions[version] {
			return nil, 0, fault.InvalidCurrencyAddress // ***** FIX THIS: is this error right?
		}
		n += paymentsLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOffset == 0 {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if err != nil {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerLength

		// signature
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		// countersignature
		countersignatureLength, countersignatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if countersignatureOffset == 0 {
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
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case BitmarkShareTag:

		// link
		linkLength, linkOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if linkOffset == 0 {
			break unpack_switch
		}
		n += linkOffset
		var link merkle.Digest
		err := merkle.DigestFromBytes(&link, record[n:n+linkLength])
		if err != nil {
			return nil, 0, err
		}
		n += linkLength

		// total number of shares to issue
		quantity, quantityLength := util.FromVarint64(record[n:])
		if quantityLength == 0 {
			break unpack_switch
		}
		n += quantityLength

		// signature
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		r := &BitmarkShare{
			Link:      link,
			Quantity:  quantity,
			Signature: signature,
		}
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case ShareGrantTag:

		// share id
		shareIdLength, shareIdOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if shareIdOffset == 0 {
			break unpack_switch
		}
		n += shareIdOffset
		var shareId merkle.Digest
		err := merkle.DigestFromBytes(&shareId, record[n:n+shareIdLength])
		if err != nil {
			return nil, 0, err
		}
		n += shareIdLength

		// number of shares to transfer
		quantity, quantityLength := util.FromVarint64(record[n:])
		if quantityLength == 0 {
			break unpack_switch
		}
		n += quantityLength

		// owner public key
		ownerLength, ownerOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOffset == 0 {
			break unpack_switch
		}
		n += ownerOffset
		owner, err := account.AccountFromBytes(record[n : n+ownerLength])
		if err != nil {
			return nil, 0, err
		}
		if owner.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerLength

		// recipient public key
		recipientLength, recipientOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if recipientOffset == 0 {
			break unpack_switch
		}
		n += recipientOffset
		recipient, err := account.AccountFromBytes(record[n : n+recipientLength])
		if err != nil {
			return nil, 0, err
		}
		if recipient.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += recipientLength

		// time limit
		beforeBlock, beforeBlockLength := util.FromVarint64(record[n:])
		if beforeBlockLength == 0 {
			break unpack_switch
		}
		n += beforeBlockLength

		// signature
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		// countersignature
		countersignatureLength, countersignatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if countersignatureOffset == 0 {
			break unpack_switch
		}
		countersignature := make(account.Signature, countersignatureLength)
		n += countersignatureOffset
		copy(countersignature, record[n:n+countersignatureLength])
		n += countersignatureLength

		r := &ShareGrant{
			ShareId:          shareId,
			Quantity:         quantity,
			Owner:            owner,
			Recipient:        recipient,
			BeforeBlock:      beforeBlock,
			Signature:        signature,
			Countersignature: countersignature,
		}
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	case ShareSwapTag:

		// share one
		shareIdOneLength, shareIdOneOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if shareIdOneOffset == 0 {
			break unpack_switch
		}
		n += shareIdOneOffset
		var shareIdOne merkle.Digest
		err := merkle.DigestFromBytes(&shareIdOne, record[n:n+shareIdOneLength])
		if err != nil {
			return nil, 0, err
		}
		n += shareIdOneLength

		// number of shares to transfer
		quantityOne, quantityOneLength := util.FromVarint64(record[n:])
		if quantityOneLength == 0 {
			break unpack_switch
		}
		n += quantityOneLength

		// owner one public key
		ownerOneLength, ownerOneOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerOneOffset == 0 {
			break unpack_switch
		}
		n += ownerOneOffset
		ownerOne, err := account.AccountFromBytes(record[n : n+ownerOneLength])
		if err != nil {
			return nil, 0, err
		}
		if ownerOne.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerOneLength

		// share two
		shareIdTwoLength, shareIdTwoOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if shareIdTwoOffset == 0 {
			break unpack_switch
		}
		n += shareIdTwoOffset
		var shareIdTwo merkle.Digest
		err = merkle.DigestFromBytes(&shareIdTwo, record[n:n+shareIdTwoLength])
		if err != nil {
			return nil, 0, err
		}
		n += shareIdTwoLength

		// number of shares to transfer
		quantityTwo, quantityTwoLength := util.FromVarint64(record[n:])
		if quantityTwoLength == 0 {
			break unpack_switch
		}
		n += quantityTwoLength

		// owner two public key
		ownerTwoLength, ownerTwoOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if ownerTwoOffset == 0 {
			break unpack_switch
		}
		n += ownerTwoOffset
		ownerTwo, err := account.AccountFromBytes(record[n : n+ownerTwoLength])
		if err != nil {
			return nil, 0, err
		}
		if ownerTwo.IsTesting() != testnet {
			return nil, 0, fault.WrongNetworkForPublicKey
		}
		n += ownerTwoLength

		// time limit
		beforeBlock, beforeBlockLength := util.FromVarint64(record[n:])
		if beforeBlockLength == 0 {
			break unpack_switch
		}
		n += beforeBlockLength

		// signature
		signatureLength, signatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if signatureOffset == 0 {
			break unpack_switch
		}
		signature := make(account.Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:n+signatureLength])
		n += signatureLength

		// countersignature
		countersignatureLength, countersignatureOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if countersignatureOffset == 0 {
			break unpack_switch
		}
		countersignature := make(account.Signature, countersignatureLength)
		n += countersignatureOffset
		copy(countersignature, record[n:n+countersignatureLength])
		n += countersignatureLength

		r := &ShareSwap{
			ShareIdOne:       shareIdOne,
			QuantityOne:      quantityOne,
			OwnerOne:         ownerOne,
			ShareIdTwo:       shareIdTwo,
			QuantityTwo:      quantityTwo,
			OwnerTwo:         ownerTwo,
			BeforeBlock:      beforeBlock,
			Signature:        signature,
			Countersignature: countersignature,
		}
		err = r.check(testnet)
		if err != nil {
			return nil, 0, err
		}
		return r, n, nil

	default: // also NullTag
	}
	return nil, 0, fault.NotTransactionPack
}

func unpackEscrow(record []byte, n int) (*Payment, int, error) {

	// optional escrow payment
	payment := (*Payment)(nil)

	switch record[n] {
	case 0:
		n += 1
	case 1:
		n += 1

		// currency
		c, currencyLength := util.FromVarint64(record[n:])
		if currencyLength == 0 {
			return nil, 0, fault.NotTransactionPack
		}
		n += currencyLength
		currencyValue, err := currency.FromUint64(c)
		if err != nil {
			return nil, 0, err
		}

		// address
		addressLength, addressOffset := util.ClippedVarint64(record[n:], 1, 8192)
		if addressOffset == 0 {
			return nil, 0, fault.NotTransactionPack
		}
		n += addressOffset
		address := string(record[n : n+addressLength])
		n += addressLength

		// amount
		amount, amountLength := util.FromVarint64(record[n:])
		if amountLength == 0 {
			return nil, 0, fault.NotTransactionPack
		}
		n += amountLength

		payment = &Payment{
			Currency: currencyValue,
			Address:  address,
			Amount:   amount,
		}
	default:
		return nil, 0, fault.NotTransactionPack
	}
	return payment, n, nil
}
