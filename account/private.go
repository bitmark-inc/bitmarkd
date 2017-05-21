// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package account

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/sha3"
)

// base type for PrivateKey
type PrivateKey struct {
	PrivateKeyInterface
}

type PrivateKeyInterface interface {
	Account() *Account
	KeyType() int
	PrivateKeyBytes() []byte
	Bytes() []byte
	String() string
	IsTesting() bool
	MarshalText() ([]byte, error)
}

// for ed25519 keys
type ED25519PrivateKey struct {
	Test       bool
	PrivateKey []byte
}

// just for debugging
type NothingPrivateKey struct {
	Test       bool
	PrivateKey []byte
}

// seed parameters
var (
	seedHeader       = []byte{0x5a, 0xfe, 0x01}
	seedHeaderLength = len(seedHeader)
	seedNonce        = [24]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	seedCountBM = [16]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0xe7,
	}
)

const (
	seedPrefixLength   = 1
	seedKeyLength      = 32
	seedChecksumLength = 4
)

// this converts a Base58 encoded seed string and returns a private key
//
// one of the specific private key types are returned using the base "PrivateKeyInterface"
// interface type to allow individual methods to be called.
func PrivateKeyFromBase58Seed(seedBase58Encoded string) (*PrivateKey, error) {

	seed := util.FromBase58(seedBase58Encoded)
	if 0 == len(seed) {
		return nil, fault.ErrCannotDecodeSeed
	}

	// Compute seed length
	keyLength := len(seed) - seedHeaderLength - seedChecksumLength
	if seedKeyLength+seedPrefixLength != keyLength {
		return nil, fault.ErrInvalidSeedLength
	}

	// Check seed header
	if !bytes.Equal(seedHeader, seed[:seedHeaderLength]) {
		return nil, fault.ErrInvalidSeedHeader
	}

	// Checksum
	checksumStart := len(seed) - checksumLength
	checksum := sha3.Sum256(seed[:checksumStart])
	if !bytes.Equal(checksum[:seedChecksumLength], seed[checksumStart:]) {
		return nil, fault.ErrChecksumMismatch
	}

	var secretKey [seedKeyLength]byte
	copy(secretKey[:], seed[seedHeaderLength+seedPrefixLength:])

	prefix := seed[seedHeaderLength : seedHeaderLength+seedPrefixLength]

	// first byte of prefix is test/live indication
	isTest := prefix[0] == 0x01

	encrypted := secretbox.Seal([]byte{}, seedCountBM[:], &seedNonce, &secretKey)

	_, priv, err := ed25519.GenerateKey(bytes.NewBuffer(encrypted))
	if nil != err {
		return nil, err
	}

	privateKey := &PrivateKey{
		PrivateKeyInterface: &ED25519PrivateKey{
			Test:       isTest,
			PrivateKey: priv,
		},
	}
	return privateKey, nil
}

// this converts a Base58 encoded string and returns an private key
//
// one of the specific private key types are returned using the base "PrivateKeyInterface"
// interface type to allow individual methods to be called.
func PrivateKeyFromBase58(privateKeyBase58Encoded string) (*PrivateKey, error) {
	// Decode the privateKey
	privateKeyDecoded := util.FromBase58(privateKeyBase58Encoded)
	if 0 == len(privateKeyDecoded) {
		return nil, fault.ErrCannotDecodePrivateKey
	}

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(privateKeyDecoded)

	// Check key type
	if 0 == keyVariantLength || keyVariant&publicKeyCode == publicKeyCode {
		return nil, fault.ErrNotPrivateKey
	}

	// compute algorithm
	keyAlgorithm := keyVariant >> algorithmShift
	if keyAlgorithm < 0 || keyAlgorithm >= algorithmLimit {
		return nil, fault.ErrInvalidKeyType
	}

	// network selection
	isTest := 0 != keyVariant&testKeyCode

	// Compute key length
	keyLength := len(privateKeyDecoded) - keyVariantLength - checksumLength
	if keyLength <= 0 {
		return nil, fault.ErrInvalidKeyLength
	}

	// Checksum
	checksumStart := len(privateKeyDecoded) - checksumLength
	checksum := sha3.Sum256(privateKeyDecoded[:checksumStart])
	if !bytes.Equal(checksum[:checksumLength], privateKeyDecoded[checksumStart:]) {
		return nil, fault.ErrChecksumMismatch
	}

	// return a pointer to the specific private key type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PrivateKeySize {
			return nil, fault.ErrInvalidKeyLength
		}
		priv := privateKeyDecoded[keyVariantLength:checksumStart]
		privateKey := &PrivateKey{
			PrivateKeyInterface: &ED25519PrivateKey{
				Test:       isTest,
				PrivateKey: priv,
			},
		}
		return privateKey, nil
	case Nothing:
		if 2 != keyLength {
			return nil, fault.ErrInvalidKeyLength
		}
		priv := privateKeyDecoded[keyVariantLength:checksumStart]
		privateKey := &PrivateKey{
			PrivateKeyInterface: &NothingPrivateKey{
				Test:       isTest,
				PrivateKey: priv,
			},
		}
		return privateKey, nil
	default:
		return nil, fault.ErrInvalidKeyType
	}
}

// this converts a byte encoded buffer and returns an private key
//
// one of the specific private key types are returned using the base "PrivateKeyInterface"
// interface type to allow individual methods to be called.
func PrivateKeyFromBytes(privateKeyBytes []byte) (*PrivateKey, error) {

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(privateKeyBytes)

	// Check key type
	if 0 == keyVariantLength || keyVariant&publicKeyCode == publicKeyCode {
		return nil, fault.ErrNotPrivateKey
	}

	// compute algorithm
	keyAlgorithm := keyVariant >> algorithmShift
	if keyAlgorithm < 0 || keyAlgorithm >= algorithmLimit {
		return nil, fault.ErrInvalidKeyType
	}

	// network selection
	isTest := 0 != keyVariant&testKeyCode

	// Compute key length
	keyLength := len(privateKeyBytes) - keyVariantLength
	if keyLength <= 0 {
		return nil, fault.ErrInvalidKeyLength
	}

	// return a pointer to the specific private key type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PrivateKeySize {
			return nil, fault.ErrInvalidKeyLength
		}
		priv := privateKeyBytes[keyVariantLength:]
		privateKey := &PrivateKey{
			PrivateKeyInterface: &ED25519PrivateKey{
				Test:       isTest,
				PrivateKey: priv,
			},
		}
		return privateKey, nil
	case Nothing:
		if 2 != keyLength {
			return nil, fault.ErrInvalidKeyLength
		}
		priv := privateKeyBytes[keyVariantLength:]
		privateKey := &PrivateKey{
			PrivateKeyInterface: &NothingPrivateKey{
				Test:       isTest,
				PrivateKey: priv,
			},
		}
		return privateKey, nil
	default:
		return nil, fault.ErrInvalidKeyType
	}
}

func (privateKey *PrivateKey) UnmarshalText(s []byte) error {
	a, err := PrivateKeyFromBase58(string(s))
	if nil != err {
		return err
	}
	privateKey.PrivateKeyInterface = a.PrivateKeyInterface
	return nil
}

// ED25519
// -------

// return whether the private key is in test mode or not
func (privateKey *ED25519PrivateKey) IsTesting() bool {
	return privateKey.Test
}

// key type code (see enumeration in account.go)
func (privateKey *ED25519PrivateKey) KeyType() int {
	return ED25519
}

// return the corresponding account
func (privateKey *ED25519PrivateKey) Account() *Account {
	return &Account{
		AccountInterface: &ED25519Account{
			Test:      privateKey.Test,
			PublicKey: privateKey.PrivateKey[ed25519.PrivateKeySize-ed25519.PublicKeySize:],
		},
	}
}

// fetch the private key as byte slice
func (privateKey *ED25519PrivateKey) PrivateKeyBytes() []byte {
	return privateKey.PrivateKey[:]
}

// byte slice for encoded key
func (privateKey *ED25519PrivateKey) Bytes() []byte {
	keyVariant := byte(ED25519 << algorithmShift)
	if privateKey.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, privateKey.PrivateKey[:]...)
}

// base58 encoding of encoded key
func (privateKey *ED25519PrivateKey) String() string {
	buffer := privateKey.Bytes()
	checksum := sha3.Sum256(buffer)
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// convert an privateKey to its Base58 JSON form
func (privateKey ED25519PrivateKey) MarshalText() ([]byte, error) {
	return []byte(privateKey.String()), nil
}

// Nothing
// -------

// return whether the private key is in test mode or not
func (privateKey *NothingPrivateKey) IsTesting() bool {
	return privateKey.Test
}

// key type code (see enumeration in account.go)
func (privateKey *NothingPrivateKey) KeyType() int {
	return Nothing
}

// return the corresponding account
func (privateKey *NothingPrivateKey) Account() *Account {
	return nil
}

// fetch the private key as byte slice
func (privateKey *NothingPrivateKey) PrivateKeyBytes() []byte {
	return privateKey.PrivateKey[:]
}

// byte slice for encoded key
func (privateKey *NothingPrivateKey) Bytes() []byte {
	keyVariant := byte(Nothing << algorithmShift)
	if privateKey.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, privateKey.PrivateKey[:]...)
}

// base58 encoding of encoded key
func (privateKey *NothingPrivateKey) String() string {
	buffer := privateKey.Bytes()
	checksum := sha3.Sum256(buffer)
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// convert an privateKey to its Base58 JSON form
func (privateKey NothingPrivateKey) MarshalText() ([]byte, error) {
	return []byte(privateKey.String()), nil
}
