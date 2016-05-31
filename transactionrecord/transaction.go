// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transactionrecord

import (
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/util"
)

type TagType uint64

// enumerate the possible transaction record types
// this is encoded a Varint64 at start of "Packed"
const (
	// null marks beginning of list - not used as a record type
	NullTag = TagType(iota)

	// valid record type
	BaseDataTag        = TagType(iota)
	AssetDataTag       = TagType(iota)
	BitmarkIssueTag    = TagType(iota)
	BitmarkTransferTag = TagType(iota)

	// this item must be last
	InvalidTag = TagType(iota)
)

// packed records are just a byte slice
type Packed []byte

// byte sizes for various fields
const (
	maxDescriptionLength    = 256
	maxNameLength           = 64
	maxFingerprintLength    = 1024
	maxSignatureLength      = 1024
	maxTimestampLength      = len("2014-06-21T14:32:16Z")
	maxPaymentAddressLength = 64
)

// the unpacked Proofer Data structure
type BaseData struct {
	Currency       currency.Currency `json:"currency"`        // utf-8 → Enum
	PaymentAddress string            `json:"payment_address"` // utf-8
	Owner          *account.Account  `json:"owner"`           // base58
	Nonce          uint64            `json:"nonce,string"`    // unsigned 0..N
	Signature      account.Signature `json:"signature,"`      // hex
}

// the unpacked Asset Data structure
type AssetData struct {
	Description string            `json:"description"` // utf-8
	Name        string            `json:"name"`        // utf-8
	Fingerprint string            `json:"fingerprint"` // utf-8
	Registrant  *account.Account  `json:"registrant"`  // base58
	Signature   account.Signature `json:"signature"`   // hex
}

// the unpacked BitmarkIssue structure
type BitmarkIssue struct {
	AssetIndex AssetIndex        `json:"asset"`     // link to asset record
	Owner      *account.Account  `json:"owner"`     // base58: the "destination" owner
	Nonce      uint64            `json:"nonce"`     // to allow for multiple issues at the same time
	Signature  account.Signature `json:"signature"` // hex: corresponds to owner in linked record
}

// optional payment record
type Payment struct {
	Currency currency.Currency `json:"currency"`      // utf-8 → Enum
	Address  string            `json:"address"`       // utf-8
	Amount   uint64            `json:"amount,string"` // number as string, interms of smallest currency unit
}

// the unpacked BitmarkTransfer structure
type BitmarkTransfer struct {
	Link      Link              `json:"link"`      // previous record
	Payment   *Payment          `json:"payment"`   // optional payment address
	Owner     *account.Account  `json:"owner"`     // base58: the "destination" owner
	Signature account.Signature `json:"signature"` // hex: corresponds to owner in linked record
}

// determine the record type code
func (record Packed) Type() TagType {
	recordType, _ := util.FromVarint64(record)
	return TagType(recordType)
}

// get the name of a transaction record as a string
func RecordName(record interface{}) (string, bool) {
	switch record.(type) {
	case *BaseData, BaseData:
		return "BaseData", true

	case *AssetData, AssetData:
		return "AssetData", true

	case *BitmarkIssue, BitmarkIssue:
		return "BitmarkIssue", true

	case *BitmarkTransfer, BitmarkTransfer:
		return "BitmarkTransfer", true

	default:
		return "*unknown*", false
	}
}

// compute an asset index
func (assetData *AssetData) AssetIndex() AssetIndex {
	return NewAssetIndex([]byte(assetData.Fingerprint))
}
