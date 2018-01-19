// Copyright (c) 2014-2018 Bitmark Inc.
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
	BitmarkTransferUnratifiedTag    = TagType(iota) // OBSOLETE: transfer
	BitmarkTransferCountersignedTag = TagType(iota) // transfer
	BlockOwnerIssueTag              = TagType(iota) // block owner
	BlockOwnerTransferTag           = TagType(iota) // block owner transfer

	// this item must be last
	InvalidTag = TagType(iota)
)

// packed records are just a byte slice
type Packed []byte

// generic transaction interface
type Transaction interface {
	Pack(account *account.Account) (Packed, error)
}

// byte sizes for various fields
const (
	minNameLength        = 1
	maxNameLength        = 64
	maxMetadataLength    = 2048
	minFingerprintLength = 1
	maxFingerprintLength = 1024
	maxSignatureLength   = 1024
	maxTimestampLength   = len("2014-06-21T14:32:16Z")
)

// the unpacked Proofer Data structure
type OldBaseData struct {
	Currency       currency.Currency `json:"currency"`       // utf-8 → Enum
	PaymentAddress string            `json:"paymentAddress"` // utf-8
	Owner          *account.Account  `json:"owner"`          // base58
	Nonce          uint64            `json:"nonce,string"`   // unsigned 0..N
	Signature      account.Signature `json:"signature,"`     // hex
}

// the unpacked Asset Data structure
type AssetData struct {
	Name        string            `json:"name"`        // utf-8
	Fingerprint string            `json:"fingerprint"` // utf-8
	Metadata    string            `json:"metadata"`    // utf-8
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
	Amount   uint64            `json:"amount,string"` // number as string, in terms of smallest currency unit
}

// a single payment possibility - for use in RPC layers
// up to entries:
//   1. issue block owner payment
//   2. last transfer block owner payment (can merge with 1 if same address)
//   3. optional transfer payment
type PaymentAlternative []*Payment

// to access field of various transfer types
type BitmarkTransfer interface {
	Transaction
	GetLink() merkle.Digest
	GetPayment() *Payment
	GetOwner() *account.Account
	GetSignature() account.Signature
	GetCountersignature() account.Signature
}

// the unpacked BitmarkTransfer structure
type BitmarkTransferUnratified struct {
	Link      merkle.Digest     `json:"link"`      // previous record
	Payment   *Payment          `json:"payment"`   // optional payment address
	Owner     *account.Account  `json:"owner"`     // base58: the "destination" owner
	Signature account.Signature `json:"signature"` // hex: corresponds to owner in linked record
}

// the unpacked Countersigned BitmarkTransfer structure
type BitmarkTransferCountersigned struct {
	Link             merkle.Digest     `json:"link"`             // previous record
	Payment          *Payment          `json:"payment"`          // optional payment address
	Owner            *account.Account  `json:"owner"`            // base58: the "destination" owner
	Signature        account.Signature `json:"signature"`        // hex: corresponds to owner in linked record
	Countersignature account.Signature `json:"countersignature"` // hex: corresponds to owner in this record
}

// the unpacked Block Owner Issue Data structure
type BlockOwnerIssue struct {
	Version   uint64            `json:"version"`      // reflects combination of supported currencies
	Payments  currency.Map      `json:"payments"`     // contents depend on version
	Owner     *account.Account  `json:"owner"`        // base58
	Nonce     uint64            `json:"nonce,string"` // unsigned 0..N
	Signature account.Signature `json:"signature,"`   // hex
}

// the unpacked Block Owner Transfer Data structure
type BlockOwnerTransfer struct {
	Link      merkle.Digest     `json:"link"`       // previous record
	Version   uint64            `json:"version"`    // reflects combination of supported currencies
	Payments  currency.Map      `json:"payments"`   // require length and contents depend on version
	Owner     *account.Account  `json:"owner"`      // base58
	Signature account.Signature `json:"signature,"` // hex
}

// determine the record type code
func (record Packed) Type() TagType {
	recordType, _ := util.FromVarint64(record)
	return TagType(recordType)
}

// get the name of a transaction record as a string
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

	case *BlockOwnerIssue, BlockOwnerIssue:
		return "BlockOwnerIssue", true

	case *BlockOwnerTransfer, BlockOwnerTransfer:
		return "BlockOwnerTransfer", true

	default:
		return "*unknown*", false
	}
}

// compute an asset index
func (assetData *AssetData) AssetIndex() AssetIndex {
	return NewAssetIndex([]byte(assetData.Fingerprint))
}

// Create an link for a packed record
func (p Packed) MakeLink() merkle.Digest {
	return merkle.NewDigest(p)
}

// convert a packed to its hex JSON form
func (p Packed) MarshalText() ([]byte, error) {
	size := hex.EncodedLen(len(p))
	b := make([]byte, size)
	hex.Encode(b, p)
	return b, nil
}

// convert a packed to its hex JSON form
func (p *Packed) UnmarshalText(s []byte) error {
	size := hex.DecodedLen(len(s))
	*p = make([]byte, size)
	_, err := hex.Decode(*p, s)
	return err
}

// to detect record types
