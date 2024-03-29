// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"strings"
	"unicode/utf8"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// supported currency sets
// code here will support all versions
var versions = []currency.Set{
	currency.MakeSet(), // 0
	currency.MakeSet(currency.Bitcoin, currency.Litecoin), // 1
}

// currently supported block foundation version (used by proofer)
const (
	FoundationVersion = 1
)

// Pack - BaseData
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (baseData *OldBaseData) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := baseData.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	err = baseData.Currency.ValidateAddress(baseData.PaymentAddress, address.IsTesting())
	if err != nil {
		return nil, err
	}

	// concatenate bytes
	message := createPacked(BaseDataTag)
	message.appendUint64(baseData.Currency.Uint64())
	message.appendString(baseData.PaymentAddress)
	message.appendAccount(baseData.Owner)
	message.appendUint64(baseData.Nonce)

	// signature
	err = address.CheckSignature(message, baseData.Signature)
	if err != nil {
		return message, err
	}
	// Signature Last
	return *message.appendBytes(baseData.Signature), nil
}

func (baseData *OldBaseData) check(testnet bool) error {
	if len(baseData.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// prevent nil or zero account
	if baseData.Owner == nil || baseData.Owner.IsZero() {
		return fault.InvalidOwnerOrRegistrant
	}

	return nil
}

// Pack - AssetData
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last.
//
// Note: the metadata field consists of key value pairs each preceded
//
//	by its count (
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (assetData *AssetData) Pack(address *account.Account) (Packed, error) {
	// prevent nil or zero account
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := assetData.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	// concatenate bytes
	message := createPacked(AssetDataTag)
	message.appendString(assetData.Name)
	message.appendString(assetData.Fingerprint)
	message.appendString(assetData.Metadata)
	message.appendAccount(assetData.Registrant)

	// signature
	err = address.CheckSignature(message, assetData.Signature)
	if err != nil {
		return message, err
	}
	// Signature Last
	return *message.appendBytes(assetData.Signature), nil
}

func (assetData *AssetData) check(testnet bool) error {
	if len(assetData.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// prevent nil or zero account
	if assetData.Registrant == nil || assetData.Registrant.IsZero() {
		return fault.InvalidOwnerOrRegistrant
	}

	if utf8.RuneCountInString(assetData.Name) > maxNameLength {
		return fault.NameTooLong
	}

	if utf8.RuneCountInString(assetData.Fingerprint) < minFingerprintLength {
		return fault.FingerprintTooShort
	}
	if utf8.RuneCountInString(assetData.Fingerprint) > maxFingerprintLength {
		return fault.FingerprintTooLong
	}

	if utf8.RuneCountInString(assetData.Metadata) > maxMetadataLength {
		return fault.MetadataTooLong
	}

	// check that metadata contains a vailid map:
	// i.e.  key1 <NUL> value1 <NUL> key2 <NUL> value2 <NUL> … keyN <NUL> valueN
	// Notes: 1: no NUL after last value
	//        2: no empty key or value is allowed
	if assetData.Metadata != "" {
		splitMetadata := strings.Split(assetData.Metadata, "\u0000")
		if len(splitMetadata)%2 == 1 {
			return fault.MetadataIsNotMap
		}
		for _, v := range splitMetadata {
			if v == "" {
				return fault.MetadataIsNotMap
			}
		}
	}
	return nil
}

// Pack - BitmarkIssue
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (issue *BitmarkIssue) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := issue.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	// concatenate bytes
	message := createPacked(BitmarkIssueTag)
	message.appendBytes(issue.AssetId[:])
	message.appendAccount(issue.Owner)
	message.appendUint64(issue.Nonce)

	// signature
	err = address.CheckSignature(message, issue.Signature)
	if err != nil {
		return message, err
	}

	// Signature Last
	return *message.appendBytes(issue.Signature), nil
}

func (issue *BitmarkIssue) check(testnet bool) error {
	if len(issue.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// prevent nil or zero account
	if issue.Owner == nil || issue.Owner.IsZero() {
		return fault.InvalidOwnerOrRegistrant
	}
	return nil
}

// Pack - BitmarkTransfer
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (transfer *BitmarkTransferUnratified) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := transfer.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	testnet := address.IsTesting()

	// concatenate bytes
	message := createPacked(BitmarkTransferUnratifiedTag)
	message.appendBytes(transfer.Link[:])
	_, err = message.appendEscrow(transfer.Escrow, testnet)
	if err != nil {
		return nil, err
	}
	message.appendAccount(transfer.Owner)

	// signature
	err = address.CheckSignature(message, transfer.Signature)
	if err != nil {
		return message, err
	}

	// Signature Last
	return *message.appendBytes(transfer.Signature), nil
}

func (transfer *BitmarkTransferUnratified) check(testnet bool) error {
	if len(transfer.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// Note: In this case Owner can be zero ⇒ bitmark is destroyed
	//       and no further transfers are allowed.
	//       the address cannot be zero to prevent discovery of the
	//       corresponding private key being able to transfer all
	//       previously destroyed bitmarks to a new account.
	if transfer.Owner == nil {
		return fault.InvalidOwnerOrRegistrant
	}

	return nil
}

// Pack - BitmarkTransferCountersigned
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (transfer *BitmarkTransferCountersigned) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := transfer.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	testnet := address.IsTesting()

	// concatenate bytes
	message := createPacked(BitmarkTransferCountersignedTag)
	message.appendBytes(transfer.Link[:])
	_, err = message.appendEscrow(transfer.Escrow, testnet)
	if err != nil {
		return nil, err
	}
	message.appendAccount(transfer.Owner)

	// signature
	err = address.CheckSignature(message, transfer.Signature)
	if err != nil {
		return message, err
	}

	// add signature Signature
	message.appendBytes(transfer.Signature)

	err = transfer.Owner.CheckSignature(message, transfer.Countersignature)
	if err != nil {
		return message, err
	}

	// Countersignature Last
	return *message.appendBytes(transfer.Countersignature), nil
}

func (transfer *BitmarkTransferCountersigned) check(testnet bool) error {
	if len(transfer.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	if len(transfer.Countersignature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// Note: impossible to have 2 signature transfer to zero public key
	if transfer.Owner == nil || transfer.Owner.IsZero() {
		return fault.InvalidOwnerOrRegistrant
	}

	return nil
}

// Pack - BlockFoundation
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (foundation *BlockFoundation) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := foundation.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	packedPayments, err := foundation.Payments.Pack(address.IsTesting())
	if err != nil {
		return nil, err
	}

	// concatenate bytes
	message := createPacked(BlockFoundationTag)
	message.appendUint64(foundation.Version)
	message.appendBytes(packedPayments)
	message.appendAccount(foundation.Owner)
	message.appendUint64(foundation.Nonce)

	// signature
	err = address.CheckSignature(message, foundation.Signature)
	if err != nil {
		return message, err
	}
	// Signature Last
	return *message.appendBytes(foundation.Signature), nil
}

func (foundation *BlockFoundation) check(testnet bool) error {
	if len(foundation.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// prevent nil or zero account
	if foundation.Owner == nil || foundation.Owner.IsZero() {
		return fault.InvalidOwnerOrRegistrant
	}

	err := CheckPayments(foundation.Version, testnet, foundation.Payments)
	if err != nil {
		return err
	}
	return nil
}

// Pack - BlockOwnerTransfer
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (transfer *BlockOwnerTransfer) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := transfer.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	packedPayments, err := transfer.Payments.Pack(address.IsTesting())
	if err != nil {
		return nil, err
	}

	testnet := address.IsTesting()

	// concatenate bytes
	message := createPacked(BlockOwnerTransferTag)
	message.appendBytes(transfer.Link[:])
	_, err = message.appendEscrow(transfer.Escrow, testnet)
	if err != nil {
		return nil, err
	}
	message.appendUint64(transfer.Version)
	message.appendBytes(packedPayments)
	message.appendAccount(transfer.Owner)

	// signature
	err = address.CheckSignature(message, transfer.Signature)
	if err != nil {
		return message, err
	}
	message.appendBytes(transfer.Signature)

	err = transfer.Owner.CheckSignature(message, transfer.Countersignature)
	if err != nil {
		return message, err
	}

	// Countersignature Last
	return *message.appendBytes(transfer.Countersignature), nil
}

func (transfer *BlockOwnerTransfer) check(testnet bool) error {
	if len(transfer.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	if len(transfer.Countersignature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// prevent nil or zero account
	if transfer.Owner == nil || transfer.Owner.IsZero() {
		return fault.InvalidOwnerOrRegistrant
	}

	err := CheckPayments(transfer.Version, testnet, transfer.Payments)
	if err != nil {
		return err
	}

	return nil
}

// Pack - BitmarkShare
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
func (share *BitmarkShare) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := share.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	// concatenate bytes
	message := createPacked(BitmarkShareTag)
	message.appendBytes(share.Link[:])
	message.appendUint64(share.Quantity)

	// signature
	err = address.CheckSignature(message, share.Signature)
	if err != nil {
		return message, err
	}
	// Signature Last
	return *message.appendBytes(share.Signature), nil
}

func (share *BitmarkShare) check(testnet bool) error {
	if len(share.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// ensure minimum share quantity
	if share.Quantity < 1 {
		return fault.ShareQuantityTooSmall
	}
	return nil
}

// Pack - ShareGrant
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
//
// NOTE: in this case address _MUST_ point to the record.Owner
func (grant *ShareGrant) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() ||
		address != grant.Owner {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := grant.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	// concatenate bytes
	message := createPacked(ShareGrantTag)
	message.appendBytes(grant.ShareId[:])
	message.appendUint64(grant.Quantity)
	message.appendAccount(grant.Owner)
	message.appendAccount(grant.Recipient)
	message.appendUint64(grant.BeforeBlock)

	// signature
	err = grant.Owner.CheckSignature(message, grant.Signature)
	if err != nil {
		return message, err
	}
	message.appendBytes(grant.Signature)

	err = grant.Recipient.CheckSignature(message, grant.Countersignature)
	if err != nil {
		return message, err
	}

	// Countersignature Last
	return *message.appendBytes(grant.Countersignature), nil
}

func (grant *ShareGrant) check(testnet bool) error {
	if len(grant.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	if len(grant.Countersignature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// prevent nil or zero account
	if grant.Owner == nil || grant.Recipient == nil ||
		grant.Owner.IsZero() || grant.Recipient.IsZero() ||
		grant.Owner == grant.Recipient {
		return fault.InvalidOwnerOrRegistrant
	}

	// ensure minimum share quantity
	if grant.Quantity < 1 {
		return fault.ShareQuantityTooSmall
	}
	return nil
}

// Pack - ShareSwap
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//
//	debugging/testing
//
// NOTE: in this case address _MUST_ point to the record.OwnerOne
func (swap *ShareSwap) Pack(address *account.Account) (Packed, error) {
	if address == nil || address.IsZero() ||
		address != swap.OwnerOne {
		return nil, fault.InvalidOwnerOrRegistrant
	}

	err := swap.check(address.IsTesting())
	if err != nil {
		return nil, err
	}

	// concatenate bytes
	message := createPacked(ShareSwapTag)
	message.appendBytes(swap.ShareIdOne[:])
	message.appendUint64(swap.QuantityOne)
	message.appendAccount(swap.OwnerOne)
	message.appendBytes(swap.ShareIdTwo[:])
	message.appendUint64(swap.QuantityTwo)
	message.appendAccount(swap.OwnerTwo)
	message.appendUint64(swap.BeforeBlock)

	// signature
	err = swap.OwnerOne.CheckSignature(message, swap.Signature)
	if err != nil {
		return message, err
	}
	message.appendBytes(swap.Signature)

	err = swap.OwnerTwo.CheckSignature(message, swap.Countersignature)
	if err != nil {
		return message, err
	}

	// Countersignature Last
	return *message.appendBytes(swap.Countersignature), nil
}

func (swap *ShareSwap) check(testnet bool) error {
	if len(swap.Signature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	if len(swap.Countersignature) > maxSignatureLength {
		return fault.SignatureTooLong
	}

	// prevent nil or zero account
	if swap.OwnerOne == nil || swap.OwnerTwo == nil ||
		swap.OwnerOne.IsZero() || swap.OwnerTwo.IsZero() ||
		swap.OwnerOne == swap.OwnerTwo {
		return fault.InvalidOwnerOrRegistrant
	}

	// ensure shares are different
	if swap.ShareIdOne == swap.ShareIdTwo {
		return fault.ShareIdsCannotBeIdentical
	}

	// ensure minimum share quantity
	if swap.QuantityOne < 1 || swap.QuantityTwo < 1 {
		return fault.ShareQuantityTooSmall
	}
	return nil
}

// internal routines below here
// ----------------------------

// CheckPayments - check all currency addresses for correct network and validity
func CheckPayments(version uint64, testnet bool, payments currency.Map) error {
	// validate version
	if version < 1 || version >= uint64(len(versions)) {
		return fault.InvalidPaymentVersion
	}

	cs := currency.MakeSet()
	for currency, address := range payments {

		err := currency.ValidateAddress(address, testnet)
		if err != nil {
			return err
		}

		// if a duplicate currency value
		if cs.Add(currency) {
			return fault.InvalidCurrencyAddress
		}
	}

	// validate the set of supplied currencies
	if versions[version] != cs {
		return fault.InvalidCurrencyAddress
	}

	return nil
}

// create a new packed buffer
func createPacked(tag TagType) Packed {
	return util.ToVarint64(uint64(tag))
}

// append a single field to a buffer
//
// the field is prefixed by Varint64(length)
func (buffer *Packed) appendString(s string) *Packed {
	l := util.ToVarint64(uint64(len(s)))
	*buffer = append(*buffer, l...)
	*buffer = append(*buffer, s...)
	return buffer
}

// append an address to a buffer
//
// the field is prefixed by Varint64(length)
func (buffer *Packed) appendAccount(address *account.Account) *Packed {
	data := address.Bytes()
	l := util.ToVarint64(uint64(len(data)))
	*buffer = append(*buffer, l...)
	*buffer = append(*buffer, data...)
	return buffer
}

// append a bytes to a buffer
//
// the field is prefixed by Varint64(length)
func (buffer *Packed) appendBytes(data []byte) *Packed {
	l := util.ToVarint64(uint64(len(data)))
	*buffer = append(*buffer, l...)
	*buffer = append(*buffer, data...)
	return buffer
}

// append a Varint64 to buffer
func (buffer *Packed) appendUint64(value uint64) *Packed {
	valueBytes := util.ToVarint64(value)
	*buffer = append(*buffer, valueBytes...)
	return buffer
}

// append a Escrop[ payment to buffer
func (buffer *Packed) appendEscrow(escrow *Payment, testnet bool) (*Packed, error) {

	if escrow == nil {
		*buffer = append(*buffer, 0)
	} else {
		err := escrow.Currency.ValidateAddress(escrow.Address, testnet)
		if err != nil {
			return nil, err
		}
		*buffer = append(*buffer, 1)
		buffer.appendUint64(escrow.Currency.Uint64())
		buffer.appendString(escrow.Address)
		buffer.appendUint64(escrow.Amount)
	}
	return buffer, nil
}
