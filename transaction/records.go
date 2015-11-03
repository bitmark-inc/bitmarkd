// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"unicode/utf8"
)

type TagType uint64

// enumerate the possible transaction record types
// this is encoded a Varint64 at start of "Packed"
const (
	// null marks beginning of list - not used as a record type
	NullTag = TagType(iota)

	// valid record type
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
	maxDescriptionLength = 256
	maxNameLength        = 64
	maxFingerprintLength = 1024
	maxSignatureLength   = 1024
	maxTimestampLength   = len("2014-06-21T14:32:16Z")
)

// the unpacked Asset Data structure
type AssetData struct {
	Description string    `json:"description"` // utf-8
	Name        string    `json:"name"`        // utf-8
	Fingerprint string    `json:"fingerprint"` // utf-8 / hex / base64
	Registrant  *Address  `json:"registrant"`  // base58
	Signature   Signature `json:"signature"`   // base64
}

// the unpacked BitmarkIssue structure
type BitmarkIssue struct {
	AssetIndex AssetIndex `json:"asset"`     // previous record (or RegistrationTransfer if first record)
	Owner      *Address   `json:"owner"`     // base58: the "destination" owner
	Nonce      uint64     `json:"nonce"`     // to allow for multiple issues at the same time
	Signature  Signature  `json:"signature"` // base64: corresponds to owner in linked record
}

// the unpacked BitmarkTransfer structure
type BitmarkTransfer struct {
	Link      Link      `json:"link"`      // previous record (or RegistrationTransfer if first record)
	Owner     *Address  `json:"owner"`     // base58: the "destination" owner
	Signature Signature `json:"signature"` // base64: corresponds to owner in linked record
}

// determine the record type code
func (record Packed) Type() TagType {
	recordType, _ := util.FromVarint64(record)
	return TagType(recordType)
}

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

	case AssetDataTag:

		// description
		descriptionLength, descriptionOffset := util.FromVarint64(record[n:])
		description := make(Packed, descriptionLength)
		n += descriptionOffset
		copy(description, record[n:])
		n += int(descriptionLength)

		// name
		nameLength, nameOffset := util.FromVarint64(record[n:])
		name := make(Packed, nameLength)
		n += nameOffset
		copy(name, record[n:])
		n += int(nameLength)

		// fingerprint
		fingerprintLength, fingerprintOffset := util.FromVarint64(record[n:])
		fingerprint := make(Packed, fingerprintLength)
		n += fingerprintOffset
		copy(fingerprint, record[n:])
		n += int(fingerprintLength)

		// registrant public key
		registrantLength, registrantOffset := util.FromVarint64(record[n:])
		n += registrantOffset
		registrant, err := AddressFromBytes(record[n : n+int(registrantLength)])
		if nil != err {
			return nil, err
		}
		n += int(registrantLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(Signature, signatureLength)
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
		owner, err := AddressFromBytes(record[n : n+int(ownerLength)])
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
		signature := make(Signature, signatureLength)
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
		var link Link
		err := LinkFromBytes(&link, record[n:n+int(linkLength)])
		if nil != err {
			return nil, err
		}
		n += int(linkLength)

		// owner public key
		ownerLength, ownerOffset := util.FromVarint64(record[n:])
		n += ownerOffset
		owner, err := AddressFromBytes(record[n : n+int(ownerLength)])
		if nil != err {
			return nil, err
		}
		n += int(ownerLength)

		// signature is remainder of record
		signatureLength, signatureOffset := util.FromVarint64(record[n:])
		signature := make(Signature, signatureLength)
		n += signatureOffset
		copy(signature, record[n:])
		n += int(signatureLength)

		r := &BitmarkTransfer{
			Link:      link,
			Owner:     owner,
			Signature: signature,
		}
		return r, nil

	default:
	}
	return nil, fault.ErrNotTransactionPack
}

// compute an asset index
func (assetData *AssetData) AssetIndex() AssetIndex {
	return NewAssetIndex([]byte(assetData.Fingerprint))
}

// pack AssetData
//
// Pack Varint64(tag) followed by fields in order as struct above with
// signature last
//
// NOTE: returns the "unsigned" message on signature failure - for
//       debugging/testing
func (assetData *AssetData) Pack(address *Address) (Packed, error) {
	if len(assetData.Signature) > maxSignatureLength {
		return nil, fault.ErrSignatureTooLong
	}

	if utf8.RuneCountInString(assetData.Description) > maxDescriptionLength {
		return nil, fault.ErrDescriptionTooLong
	}

	if utf8.RuneCountInString(assetData.Name) > maxNameLength {
		return nil, fault.ErrNameTooLong
	}

	if utf8.RuneCountInString(assetData.Fingerprint) > maxFingerprintLength {
		return nil, fault.ErrFingerprintTooLong
	}

	// concatenate bytes
	message := util.ToVarint64(uint64(AssetDataTag))
	message = appendString(message, assetData.Description)
	message = appendString(message, assetData.Name)
	message = appendString(message, assetData.Fingerprint)
	message = appendAddress(message, assetData.Registrant)

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
func (issue *BitmarkIssue) Pack(address *Address) (Packed, error) {
	if len(issue.Signature) > maxSignatureLength {
		return nil, fault.ErrSignatureTooLong
	}

	// concatenate bytes
	message := util.ToVarint64(uint64(BitmarkIssueTag))
	message = appendBytes(message, issue.AssetIndex.Bytes())
	message = appendAddress(message, issue.Owner)
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
func (transfer *BitmarkTransfer) Pack(address *Address) (Packed, error) {
	if len(transfer.Signature) > maxSignatureLength {
		return nil, fault.ErrSignatureTooLong
	}

	// concatenate bytes
	message := util.ToVarint64(uint64(BitmarkTransferTag))
	message = appendBytes(message, transfer.Link.Bytes())
	message = appendAddress(message, transfer.Owner)

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

// append a address to a buffer
//
// the field is prefixed by Varint64(length)
func appendAddress(buffer Packed, address *Address) Packed {
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

// append a a Varint64 to buffer
func appendUint64(buffer Packed, value uint64) Packed {
	valueBytes := util.ToVarint64(value)
	buffer = append(buffer, valueBytes...)
	return buffer
}

// convert a packed to its hex JSON form
func (p Packed) MarshalJSON() ([]byte, error) {
	size := 2 + hex.EncodedLen(len(p))
	b := make([]byte, size)
	b[0] = '"'
	b[size-1] = '"'
	hex.Encode(b[1:], p)
	return b, nil
}
