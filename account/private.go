// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package account

import (
	"bytes"

	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

// PrivateKey - base type for PrivateKey
type PrivateKey struct {
	PrivateKeyInterface
}

// PrivateKeyInterface - interface type for private key methods
type PrivateKeyInterface interface {
	Account() *Account
	KeyType() int
	PrivateKeyBytes() []byte
	Bytes() []byte
	String() string
	IsTesting() bool
	MarshalText() ([]byte, error)
}

// ED25519PrivateKey - structure for ed25519 keys
type ED25519PrivateKey struct {
	Test       bool
	PrivateKey []byte
}

// NothingPrivateKey - just for debugging
type NothingPrivateKey struct {
	Test       bool
	PrivateKey []byte
}

// PrivateKeyFromBase58 - this converts a Base58 encoded string and returns an private key
//
// one of the specific private key types are returned using the base "PrivateKeyInterface"
// interface type to allow individual methods to be called.
func PrivateKeyFromBase58(privateKeyBase58Encoded string) (*PrivateKey, error) {
	// Decode the privateKey
	privateKeyDecoded := util.FromBase58(privateKeyBase58Encoded)
	if 0 == len(privateKeyDecoded) {
		return nil, fault.CannotDecodePrivateKey
	}

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(privateKeyDecoded)

	// Check key type
	if 0 == keyVariantLength || keyVariant&publicKeyCode == publicKeyCode {
		return nil, fault.NotPrivateKey
	}

	// compute algorithm
	keyAlgorithm := keyVariant >> algorithmShift
	if keyAlgorithm >= algorithmLimit {
		return nil, fault.InvalidKeyType
	}

	// network selection
	isTest := 0 != keyVariant&testKeyCode

	// Compute key length
	keyLength := len(privateKeyDecoded) - keyVariantLength - checksumLength
	if keyLength <= 0 {
		return nil, fault.InvalidKeyLength
	}

	// Checksum
	checksumStart := len(privateKeyDecoded) - checksumLength
	checksum := sha3.Sum256(privateKeyDecoded[:checksumStart])
	if !bytes.Equal(checksum[:checksumLength], privateKeyDecoded[checksumStart:]) {
		return nil, fault.ChecksumMismatch
	}

	// return a pointer to the specific private key type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PrivateKeySize {
			return nil, fault.InvalidKeyLength
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
			return nil, fault.InvalidKeyLength
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
		return nil, fault.InvalidKeyType
	}
}

// PrivateKeyFromBytes - this converts a byte encoded buffer and returns an private key
//
// one of the specific private key types are returned using the base "PrivateKeyInterface"
// interface type to allow individual methods to be called.
func PrivateKeyFromBytes(privateKeyBytes []byte) (*PrivateKey, error) {

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(privateKeyBytes)

	// Check key type
	if 0 == keyVariantLength || keyVariant&publicKeyCode == publicKeyCode {
		return nil, fault.NotPrivateKey
	}

	// compute algorithm
	keyAlgorithm := keyVariant >> algorithmShift
	if keyAlgorithm >= algorithmLimit {
		return nil, fault.InvalidKeyType
	}

	// network selection
	isTest := 0 != keyVariant&testKeyCode

	// Compute key length
	keyLength := len(privateKeyBytes) - keyVariantLength
	if keyLength <= 0 {
		return nil, fault.InvalidKeyLength
	}

	// return a pointer to the specific private key type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PrivateKeySize {
			return nil, fault.InvalidKeyLength
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
			return nil, fault.InvalidKeyLength
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
		return nil, fault.InvalidKeyType
	}
}

// UnmarshalText - convert string to private key structure
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

// IsTesting - return whether the private key is in test mode or not
func (privateKey *ED25519PrivateKey) IsTesting() bool {
	return privateKey.Test
}

// KeyType - key type code (see enumeration in account.go)
func (privateKey *ED25519PrivateKey) KeyType() int {
	return ED25519
}

// Account - return the corresponding account
func (privateKey *ED25519PrivateKey) Account() *Account {
	return &Account{
		AccountInterface: &ED25519Account{
			Test:      privateKey.Test,
			PublicKey: privateKey.PrivateKey[ed25519.PrivateKeySize-ed25519.PublicKeySize:],
		},
	}
}

// PrivateKeyBytes - fetch the private key as byte slice
func (privateKey *ED25519PrivateKey) PrivateKeyBytes() []byte {
	return privateKey.PrivateKey[:]
}

// Bytes - byte slice for encoded key
func (privateKey *ED25519PrivateKey) Bytes() []byte {
	keyVariant := byte(ED25519 << algorithmShift)
	if privateKey.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, privateKey.PrivateKey[:]...)
}

// String - base58 encoding of encoded key
func (privateKey *ED25519PrivateKey) String() string {
	buffer := privateKey.Bytes()
	checksum := sha3.Sum256(buffer)
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// MarshalText - convert an privateKey to its Base58 JSON form
func (privateKey ED25519PrivateKey) MarshalText() ([]byte, error) {
	return []byte(privateKey.String()), nil
}

// Nothing
// -------

// IsTesting - return whether the private key is in test mode or not
func (privateKey *NothingPrivateKey) IsTesting() bool {
	return privateKey.Test
}

// KeyType - key type code (see enumeration in account.go)
func (privateKey *NothingPrivateKey) KeyType() int {
	return Nothing
}

// Account - return the corresponding account
func (privateKey *NothingPrivateKey) Account() *Account {
	return nil
}

// PrivateKeyBytes - fetch the private key as byte slice
func (privateKey *NothingPrivateKey) PrivateKeyBytes() []byte {
	return privateKey.PrivateKey[:]
}

// Bytes - byte slice for encoded key
func (privateKey *NothingPrivateKey) Bytes() []byte {
	keyVariant := byte(Nothing << algorithmShift)
	if privateKey.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, privateKey.PrivateKey[:]...)
}

// String - base58 encoding of encoded key
func (privateKey *NothingPrivateKey) String() string {
	buffer := privateKey.Bytes()
	checksum := sha3.Sum256(buffer)
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// MarshalText - convert an privateKey to its Base58 JSON form
func (privateKey NothingPrivateKey) MarshalText() ([]byte, error) {
	return []byte(privateKey.String()), nil
}
