// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction

import (
	"bytes"
	"crypto/sha256"
	"github.com/agl/ed25519"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
)

// enumeration of supported key algorithms
const (
	// list of valid algorithms
	Nothing = iota // zero keytype **Just for Testing**
	ED25519 = iota
	// end of list (one greater than last item)
	algorithmLimit = iota
)

// miscellaneous constants
const (
	checksumLength = 4

	// bits in key code starting from LSB
	publicKeyCode = 0x01
	testKeyCode   = 0x02
	spare1KeyCode = 0x04
	spare2KeyCode = 0x08

	algorithmShift = 4 // shift 4 bits to get algorithm

)

// base type for addresses
type Address struct {
	AddressInterface
}

type AddressInterface interface {
	KeyType() int
	PublicKeyBytes() []byte
	CheckSignature(message []byte, signature Signature) error
	Bytes() []byte
	String() string
	MarshalJSON() ([]byte, error)
	//UnmarshalJSON([]byte) error
}

// for ed25519 signatures
type ED25519Address struct {
	Test      bool
	PublicKey *[ed25519.PublicKeySize]byte
}

// just for debugging
type NothingAddress struct {
	Test      bool
	PublicKey *[2]byte
}

// this converts a Base58 encoded string and returns an address
//
// one of the specific address types are returned using the base "AddressInterface"
// interface type to allow individual methods to be called.
func AddressFromBase58(addressBase58Encoded string) (*Address, error) {
	// Decode the address
	addressDecoded := util.FromBase58(addressBase58Encoded)
	if 0 == len(addressDecoded) {
		return nil, fault.ErrCannotDecodeAddress
	}

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(addressDecoded)

	// Check key type
	if 0 == keyVariantLength || keyVariant&publicKeyCode != publicKeyCode {
		return nil, fault.ErrNotPublicKey
	}

	// compute algorithm
	keyAlgorithm := keyVariant >> algorithmShift
	if keyAlgorithm < 0 || keyAlgorithm >= algorithmLimit {
		return nil, fault.ErrInvalidKeyType
	}

	// network selection
	isTest := 0 != keyVariant&testKeyCode
	if mode.IsTesting() != isTest {
		return nil, fault.ErrWrongNetworkForPublicKey
	}

	// Compute key length
	keyLength := len(addressDecoded) - keyVariantLength - checksumLength
	if keyLength <= 0 {
		return nil, fault.ErrInvalidKeyLength
	}

	// Checksum
	checksumStart := len(addressDecoded) - checksumLength
	firstRound := sha256.Sum256(addressDecoded[:checksumStart])
	checksum := sha256.Sum256(firstRound[:])
	if !bytes.Equal(checksum[:checksumLength], addressDecoded[checksumStart:]) {
		return nil, fault.ErrChecksumMismatch
	}

	// return a pointer to the specific address type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PublicKeySize {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := [ed25519.PublicKeySize]byte{}
		copy(publicKey[:], addressDecoded[keyVariantLength:checksumStart])
		address := &Address{
			AddressInterface: &ED25519Address{
				Test:      isTest,
				PublicKey: &publicKey,
			},
		}
		return address, nil
	case Nothing:
		if 2 != keyLength {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := [2]byte{}
		copy(publicKey[:], addressDecoded[keyVariantLength:checksumStart])
		address := &Address{
			AddressInterface: &NothingAddress{
				Test:      isTest,
				PublicKey: &publicKey,
			},
		}
		return address, nil
	default:
		return nil, fault.ErrInvalidKeyType
	}
}

// this converts a byte encoded buffer and returns an address
//
// one of the specific address types are returned using the base "AddressInterface"
// interface type to allow individual methods to be called.
func AddressFromBytes(addressBytes []byte) (*Address, error) {

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(addressBytes)

	// Check key type
	if 0 == keyVariantLength || keyVariant&publicKeyCode != publicKeyCode {
		return nil, fault.ErrNotPublicKey
	}

	// compute algorithm
	keyAlgorithm := keyVariant >> algorithmShift
	if keyAlgorithm < 0 || keyAlgorithm >= algorithmLimit {
		return nil, fault.ErrInvalidKeyType
	}

	// network selection
	isTest := 0 != keyVariant&testKeyCode
	if mode.IsTesting() != isTest {
		return nil, fault.ErrWrongNetworkForPublicKey
	}

	// Compute key length
	keyLength := len(addressBytes) - keyVariantLength
	if keyLength <= 0 {
		return nil, fault.ErrInvalidKeyLength
	}

	// return a pointer to the specific address type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PublicKeySize {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := [ed25519.PublicKeySize]byte{}
		copy(publicKey[:], addressBytes[keyVariantLength:])
		address := &Address{
			AddressInterface: &ED25519Address{
				Test:      isTest,
				PublicKey: &publicKey,
			},
		}
		return address, nil
	case Nothing:
		if 2 != keyLength {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := [2]byte{}
		copy(publicKey[:], addressBytes[keyVariantLength:])
		address := &Address{
			AddressInterface: &NothingAddress{
				Test:      isTest,
				PublicKey: &publicKey,
			},
		}
		return address, nil
	default:
		return nil, fault.ErrInvalidKeyType
	}
}

// convert an address from its Base58 JSON form to binary
//
// this cannot be forwarded by go compiler, since it needs to determine
// the the resulting interface type from the encoded data
func (address *Address) UnmarshalJSON(s []byte) error {
	// length = '"' + Base58 characters + '"'
	if '"' != s[0] || '"' != s[len(s)-1] {
		return fault.ErrInvalidCharacter
	}
	a, err := AddressFromBase58(string(s[1 : len(s)-1]))
	if nil != err {
		return err
	}
	address.AddressInterface = a.AddressInterface
	return nil
}

// ED25519
// -------

// key type code (see enumeration above)
func (address *ED25519Address) KeyType() int {
	return ED25519
}

// fetch the public key as byte slice
func (address *ED25519Address) PublicKeyBytes() []byte {
	return address.PublicKey[:]
}

// check the signature of a message
func (address *ED25519Address) CheckSignature(message []byte, signature Signature) error {

	if ed25519.SignatureSize != len(signature) {
		return fault.ErrInvalidSignature
	}

	// ***** FIX THIS: any way to avoid these exta copies *****
	s := [ed25519.SignatureSize]byte{}
	copy(s[:], signature[:])

	if !ed25519.Verify(address.PublicKey, message, &s) {
		return fault.ErrInvalidSignature
	}
	return nil
}

// byte slice for encoded key
func (address *ED25519Address) Bytes() []byte {
	keyVariant := byte(ED25519<<algorithmShift) | publicKeyCode
	if address.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, address.PublicKey[:]...)
}

// base58 encoding of encoded key
func (address *ED25519Address) String() string {
	buffer := address.Bytes()
	firstRound := sha256.Sum256(buffer)
	checksum := sha256.Sum256(firstRound[:])
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// convert an address to its Base58 JSON form
func (address ED25519Address) MarshalJSON() ([]byte, error) {
	b := make([]byte, 1)
	b[0] = '"'
	b = append(b, address.String()...)
	b = append(b, '"')
	return b, nil
}

// Nothing
// -------

// key type code (see enumeration above)
func (address *NothingAddress) KeyType() int {
	return Nothing
}

// fetch the public key as byte slice
func (address *NothingAddress) PublicKeyBytes() []byte {
	return address.PublicKey[:]
}

// check the signature of a message
func (address *NothingAddress) CheckSignature(message []byte, signature Signature) error {
	return fault.ErrInvalidSignature
}

// byte slice for encoded key
func (address *NothingAddress) Bytes() []byte {
	keyVariant := byte(Nothing<<algorithmShift) | publicKeyCode
	if address.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, address.PublicKey[:]...)
}

// base58 encoding of encoded key
func (address *NothingAddress) String() string {
	buffer := address.Bytes()
	firstRound := sha256.Sum256(buffer)
	checksum := sha256.Sum256(firstRound[:])
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// convert an address to its Base58 JSON form
func (address NothingAddress) MarshalJSON() ([]byte, error) {
	b := make([]byte, 32)
	b[0] = '"'
	b = append(b, address.String()...)
	b[len(b)] = '"'
	return b, nil
}
