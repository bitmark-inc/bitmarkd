// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"strings"
	"unicode/utf8"
)

// pack BaseData
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//       debugging/testing
func (baseData *BaseData) Pack(address *account.Account) (Packed, error) {
	if len(baseData.Signature) > maxSignatureLength {
		return nil, fault.ErrSignatureTooLong
	}

	if nil == baseData.Owner || nil == address {
		return nil, fault.ErrInvalidOwnerOrRegistrant
	}

	if utf8.RuneCountInString(baseData.PaymentAddress) > maxPaymentAddressLength {
		return nil, fault.ErrPaymentAddressTooLong
	}

	// concatenate bytes
	message := util.ToVarint64(uint64(BaseDataTag))
	message = appendUint64(message, baseData.Currency.Uint64())
	message = appendString(message, baseData.PaymentAddress)
	message = appendAccount(message, baseData.Owner)
	message = appendUint64(message, baseData.Nonce)

	// signature
	err := address.CheckSignature(message, baseData.Signature)
	if nil != err {
		return message, err
	}
	// Signature Last
	return appendBytes(message, baseData.Signature), nil
}

// pack AssetData
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last.
//
// Note: the metadata field consists of key value pairs each preceded
//       by its count (
//
// NOTE: returns the "unsigned" message on signature failure - for
//       debugging/testing
func (assetData *AssetData) Pack(address *account.Account) (Packed, error) {
	if len(assetData.Signature) > maxSignatureLength {
		return nil, fault.ErrSignatureTooLong
	}
	if nil == assetData.Registrant || nil == address {
		return nil, fault.ErrInvalidOwnerOrRegistrant
	}

	if utf8.RuneCountInString(assetData.Name) < minNameLength {
		return nil, fault.ErrNameTooShort
	}
	if utf8.RuneCountInString(assetData.Name) > maxNameLength {
		return nil, fault.ErrNameTooLong
	}

	if utf8.RuneCountInString(assetData.Fingerprint) < minFingerprintLength {
		return nil, fault.ErrFingerprintTooShort
	}
	if utf8.RuneCountInString(assetData.Fingerprint) > maxFingerprintLength {
		return nil, fault.ErrFingerprintTooLong
	}

	if utf8.RuneCountInString(assetData.Metadata) > maxMetadataLength {
		return nil, fault.ErrMetadataTooLong
	}

	// check that metadata contains a vailid map:
	// i.e.  key1 <NUL> value1 <NUL> key2 <NUL> value2 <NUL> … keyN <NUL> valueN
	// Notes: 1: no NUL after last value
	//        2: no empty key or value is allowed
	if 0 != len(assetData.Metadata) {
		splitMetadata := strings.Split(assetData.Metadata, "\u0000")
		if 1 == len(splitMetadata)%2 {
			return nil, fault.ErrMetadataIsNotMap
		}
		for _, v := range splitMetadata {
			if 0 == len(v) {
				return nil, fault.ErrMetadataIsNotMap
			}
		}
	}

	// concatenate bytes
	message := util.ToVarint64(uint64(AssetDataTag))
	message = appendString(message, assetData.Name)
	message = appendString(message, assetData.Fingerprint)
	message = appendString(message, assetData.Metadata)
	message = appendAccount(message, assetData.Registrant)

	// signature
	err := address.CheckSignature(message, assetData.Signature)
	if nil != err {
		return message, err
	}
	// Signature Last
	return appendBytes(message, assetData.Signature), nil
}

// pack BitmarkIssue
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//       debugging/testing
func (issue *BitmarkIssue) Pack(address *account.Account) (Packed, error) {
	if len(issue.Signature) > maxSignatureLength {
		return nil, fault.ErrSignatureTooLong
	}

	if nil == issue.Owner || nil == address {
		return nil, fault.ErrInvalidOwnerOrRegistrant
	}

	// concatenate bytes
	message := util.ToVarint64(uint64(BitmarkIssueTag))
	message = appendBytes(message, issue.AssetIndex[:])
	message = appendAccount(message, issue.Owner)
	message = appendUint64(message, issue.Nonce)

	// signature
	err := address.CheckSignature(message, issue.Signature)
	if nil != err {
		return message, err
	}

	// Signature Last
	return appendBytes(message, issue.Signature), nil
}

// local function to pack BitmarkTransfer
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//       debugging/testing
func (transfer *BitmarkTransfer) Pack(address *account.Account) (Packed, error) {
	if len(transfer.Signature) > maxSignatureLength {
		return nil, fault.ErrSignatureTooLong
	}

	if nil == transfer.Owner || nil == address {
		return nil, fault.ErrInvalidOwnerOrRegistrant
	}

	// concatenate bytes
	message := util.ToVarint64(uint64(BitmarkTransferTag))
	message = appendBytes(message, transfer.Link[:])

	if nil == transfer.Payment {
		message = append(message, 0)
	} else {
		if utf8.RuneCountInString(transfer.Payment.Address) > maxPaymentAddressLength {
			return nil, fault.ErrPaymentAddressTooLong
		}
		message = append(message, 1)
		message = appendUint64(message, transfer.Payment.Currency.Uint64())
		message = appendString(message, transfer.Payment.Address)
		message = appendUint64(message, transfer.Payment.Amount)
	}

	message = appendAccount(message, transfer.Owner)

	// signature
	err := address.CheckSignature(message, transfer.Signature)
	if nil != err {
		return message, err
	}

	// Signature Last
	return appendBytes(message, transfer.Signature), nil
}

// append a single field to a buffer
//
// the field is prefixed by Varint64(length)
func appendString(buffer Packed, s string) Packed {
	l := util.ToVarint64(uint64(len(s)))
	buffer = append(buffer, l...)
	return append(buffer, s...)
}

// append an address to a buffer
//
// the field is prefixed by Varint64(length)
func appendAccount(buffer Packed, address *account.Account) Packed {
	data := address.Bytes()
	l := util.ToVarint64(uint64(len(data)))
	buffer = append(buffer, l...)
	buffer = append(buffer, data...)
	return buffer
}

// append a bytes to a buffer
//
// the field is prefixed by Varint64(length)
func appendBytes(buffer Packed, data []byte) Packed {
	l := util.ToVarint64(uint64(len(data)))
	buffer = append(buffer, l...)
	buffer = append(buffer, data...)
	return buffer
}

// append a Varint64 to buffer
func appendUint64(buffer Packed, value uint64) Packed {
	valueBytes := util.ToVarint64(value)
	buffer = append(buffer, valueBytes...)
	return buffer
}
