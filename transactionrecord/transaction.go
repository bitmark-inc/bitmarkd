// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"encoding/hex"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/merkle"
	"github.com/bitmark-inc/bitmarkd/util"
)

// TagType - type code for transactions
type TagType uint64

// enumerate the possible transaction record types
// this is encoded a Varint64 at start of "Packed"
const (
	// null marks beginning of list - not used as a record type
	NullTag = TagType(iota)

	// valid record types
	// OBSOLETE items must still be supported to process older blocks
	BaseDataTag                     = TagType(iota) // OBSOLETE: block owner
	AssetDataTag                    = TagType(iota) // create asset
	BitmarkIssueTag                 = TagType(iota) // issue asset
	BitmarkTransferUnratifiedTag    = TagType(iota) // single signed transfer
	BitmarkTransferCountersignedTag = TagType(iota) // two signature transfer
	BlockFoundationTag              = TagType(iota) // block owner
	BlockOwnerTransferTag           = TagType(iota) // block owner transfer
	BitmarkShareTag                 = TagType(iota) // convert bitmark to a quantity of shares
	ShareGrantTag                   = TagType(iota) // grant some value to another account
	ShareSwapTag                    = TagType(iota) // atomically swap shares between accounts

	// this item must be last
	InvalidTag = TagType(iota)
)

// Packed - packed records are just a byte slice
type Packed []byte

// Transaction - generic transaction interface
type Transaction interface {
	Pack(account *account.Account) (Packed, error)
}

// byte sizes for various fields
const (
	maxNameLength        = 64
	maxMetadataLength    = 2048
	minFingerprintLength = 1
	maxFingerprintLength = 1024
	maxSignatureLength   = 1024
)

// OldBaseData - the unpacked Proofer Data structure (OBSOLETE)
// this is first tx in every block and can only be used there
type OldBaseData struct {
	Currency       currency.Currency `json:"currency"`       // utf-8 → Enum
	PaymentAddress string            `json:"paymentAddress"` // utf-8
	Owner          *account.Account  `json:"owner"`          // base58
	Nonce          uint64            `json:"nonce,string"`   // unsigned 0..N
	Signature      account.Signature `json:"signature,"`     // hex
}

// AssetData - the unpacked Asset Data structure
type AssetData struct {
	Name        string            `json:"name"`        // utf-8
	Fingerprint string            `json:"fingerprint"` // utf-8
	Metadata    string            `json:"metadata"`    // utf-8
	Registrant  *account.Account  `json:"registrant"`  // base58
	Signature   account.Signature `json:"signature"`   // hex
}

// BitmarkIssue - the unpacked BitmarkIssue structure
type BitmarkIssue struct {
	AssetId   AssetIdentifier   `json:"assetId"`   // link to asset record
	Owner     *account.Account  `json:"owner"`     // base58: the "destination" owner
	Nonce     uint64            `json:"nonce"`     // to allow for multiple issues at the same time
	Signature account.Signature `json:"signature"` // hex: corresponds to owner in linked record
}

// Payment - optional payment record
type Payment struct {
	Currency currency.Currency `json:"currency"`      // utf-8 → Enum
	Address  string            `json:"address"`       // utf-8
	Amount   uint64            `json:"amount,string"` // number as string, in terms of smallest currency unit
}

// PaymentAlternative - a single payment possibility - for use in RPC layers
// up to entries:
//   1. issue block owner payment
//   2. last transfer block owner payment (can merge with 1 if same address)
//   3. optional transfer payment
type PaymentAlternative []*Payment

// BitmarkTransfer - to access field of various transfer types
type BitmarkTransfer interface {
	Transaction
	GetLink() merkle.Digest
	GetPayment() *Payment
	GetOwner() *account.Account
	GetCurrencies() currency.Map
	GetSignature() account.Signature
	GetCountersignature() account.Signature
}

// BitmarkTransferUnratified - the unpacked BitmarkTransfer structure
type BitmarkTransferUnratified struct {
	Link      merkle.Digest     `json:"link"`      // previous record
	Escrow    *Payment          `json:"escrow"`    // optional escrow payment address
	Owner     *account.Account  `json:"owner"`     // base58: the "destination" owner
	Signature account.Signature `json:"signature"` // hex: corresponds to owner in linked record
}

// BitmarkTransferCountersigned - the unpacked Countersigned BitmarkTransfer structure
type BitmarkTransferCountersigned struct {
	Link             merkle.Digest     `json:"link"`             // previous record
	Escrow           *Payment          `json:"escrow"`           // optional escrow payment address
	Owner            *account.Account  `json:"owner"`            // base58: the "destination" owner
	Signature        account.Signature `json:"signature"`        // hex: corresponds to owner in linked record
	Countersignature account.Signature `json:"countersignature"` // hex: corresponds to owner in this record
}

// BlockFoundation - the unpacked Proofer Data structure
// this is first tx in every block and can only be used there
type BlockFoundation struct {
	Version   uint64            `json:"version"`      // reflects combination of supported currencies
	Payments  currency.Map      `json:"payments"`     // contents depend on version
	Owner     *account.Account  `json:"owner"`        // base58
	Nonce     uint64            `json:"nonce,string"` // unsigned 0..N
	Signature account.Signature `json:"signature"`    // hex
}

// BlockOwnerTransfer - the unpacked Block Owner Transfer Data structure
// forms a chain that links back to a foundation record which has a TxId of:
// SHA3-256 . concat blockDigest leBlockNumberUint64
type BlockOwnerTransfer struct {
	Link             merkle.Digest     `json:"link"`             // previous record
	Escrow           *Payment          `json:"escrow"`           // optional escrow payment address
	Version          uint64            `json:"version"`          // reflects combination of supported currencies
	Payments         currency.Map      `json:"payments"`         // require length and contents depend on version
	Owner            *account.Account  `json:"owner"`            // base58
	Signature        account.Signature `json:"signature"`        // hex
	Countersignature account.Signature `json:"countersignature"` // hex: corresponds to owner in this record
}

// BitmarkShare - turn a bitmark provenance chain into a fungible share
type BitmarkShare struct {
	Link      merkle.Digest     `json:"link"`      // previous record
	Quantity  uint64            `json:"quantity"`  // initial balance quantity
	Signature account.Signature `json:"signature"` // hex
}

// ShareGrant - grant some shares to another (one way transfer)
type ShareGrant struct {
	ShareId          merkle.Digest     `json:"shareId"`          // share = issue id
	Quantity         uint64            `json:"quantity"`         // shares to transfer > 0
	Owner            *account.Account  `json:"owner"`            // base58
	Recipient        *account.Account  `json:"recipient"`        // base58
	BeforeBlock      uint64            `json:"beforeBlock"`      // expires when chain height > before block
	Signature        account.Signature `json:"signature"`        // hex
	Countersignature account.Signature `json:"countersignature"` // hex: corresponds to owner in this record
}

// ShareSwap - swap some shares to another (two way transfer)
type ShareSwap struct {
	ShareIdOne       merkle.Digest     `json:"shareIdOne"`       // share = issue id
	QuantityOne      uint64            `json:"quantityOne"`      // shares to transfer > 0
	OwnerOne         *account.Account  `json:"ownerOne"`         // base58
	ShareIdTwo       merkle.Digest     `json:"shareIdTwo"`       // share = issue id
	QuantityTwo      uint64            `json:"quantityTwo"`      // shares to transfer > 0
	OwnerTwo         *account.Account  `json:"ownerTwo"`         // base58
	BeforeBlock      uint64            `json:"beforeBlock"`      // expires when chain height > before block
	Signature        account.Signature `json:"signature"`        // hex
	Countersignature account.Signature `json:"countersignature"` // hex: corresponds to owner in this record
}

// Type - returns the record type code
func (record Packed) Type() TagType {
	recordType, n := util.FromVarint64(record)
	if 0 == n {
		return NullTag
	}
	return TagType(recordType)
}

// RecordName - returns the name of a transaction record as a string
func RecordName(record interface{}) (string, bool) {
	switch record.(type) {
	case *OldBaseData, OldBaseData:
		return "BaseData", true

	case *AssetData, AssetData:
		return "AssetData", true

	case *BitmarkIssue, BitmarkIssue:
		return "BitmarkIssue", true

	case *BitmarkTransferUnratified, BitmarkTransferUnratified:
		return "BitmarkTransferUnratified", true

	case *BitmarkTransferCountersigned, BitmarkTransferCountersigned:
		return "BitmarkTransferCountersigned", true

	case *BlockFoundation, BlockFoundation:
		return "BlockFoundation", true

	case *BlockOwnerTransfer, BlockOwnerTransfer:
		return "BlockOwnerTransfer", true

	case *BitmarkShare, BitmarkShare:
		return "ShareBalance", true

	case *ShareGrant, ShareGrant:
		return "ShareGrant", true

	case *ShareSwap, ShareSwap:
		return "ShareSwap", true

	default:
		return "*unknown*", false
	}
}

// AssetId - compute an asset id
func (assetData *AssetData) AssetId() AssetIdentifier {
	return NewAssetIdentifier([]byte(assetData.Fingerprint))
}

// MakeLink - Create an link for a packed record
func (record Packed) MakeLink() merkle.Digest {
	return merkle.NewDigest(record)
}

// MarshalText - convert a packed to its hex JSON form
func (record Packed) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(record))
	b := make([]byte, size)
	hex.Encode(b, record)
	return b, nil
}

// UnmarshalText - convert a packed to its hex JSON form
func (record *Packed) UnmarshalText(s []byte) error {
	size := hex.DecodedLen(len(s))
	*record = make([]byte, size)
	_, err := hex.Decode(*record, s)
	return err
}
