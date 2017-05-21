// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package account

import (
	"bytes"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
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

// base type for accounts
type Account struct {
	AccountInterface
}

type AccountInterface interface {
	KeyType() int
	PublicKeyBytes() []byte
	CheckSignature(message []byte, signature Signature) error
	Bytes() []byte
	String() string
	MarshalText() ([]byte, error)
	IsTesting() bool
}

// for ed25519 signatures
type ED25519Account struct {
	Test      bool
	PublicKey []byte
}

// just for debugging
type NothingAccount struct {
	Test      bool
	PublicKey []byte
}

// this converts a Base58 encoded string and returns an account
//
// one of the specific account types are returned using the base "AccountInterface"
// interface type to allow individual methods to be called.
func AccountFromBase58(accountBase58Encoded string) (*Account, error) {
	// Decode the account
	accountDecoded := util.FromBase58(accountBase58Encoded)
	if 0 == len(accountDecoded) {
		return nil, fault.ErrCannotDecodeAccount
	}

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(accountDecoded)

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

	// Compute key length
	keyLength := len(accountDecoded) - keyVariantLength - checksumLength
	if keyLength <= 0 {
		return nil, fault.ErrInvalidKeyLength
	}

	// Checksum
	checksumStart := len(accountDecoded) - checksumLength
	checksum := sha3.Sum256(accountDecoded[:checksumStart])
	if !bytes.Equal(checksum[:checksumLength], accountDecoded[checksumStart:]) {
		return nil, fault.ErrChecksumMismatch
	}

	// return a pointer to the specific account type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PublicKeySize {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := accountDecoded[keyVariantLength:checksumStart]
		account := &Account{
			AccountInterface: &ED25519Account{
				Test:      isTest,
				PublicKey: publicKey,
			},
		}
		return account, nil
	case Nothing:
		if 2 != keyLength {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := accountDecoded[keyVariantLength:checksumStart]
		account := &Account{
			AccountInterface: &NothingAccount{
				Test:      isTest,
				PublicKey: publicKey,
			},
		}
		return account, nil
	default:
		return nil, fault.ErrInvalidKeyType
	}
}

// this converts a byte encoded buffer and returns an account
//
// one of the specific account types are returned using the base "AccountInterface"
// interface type to allow individual methods to be called.
func AccountFromBytes(accountBytes []byte) (*Account, error) {

	// Parse the key variant
	keyVariant, keyVariantLength := util.FromVarint64(accountBytes)

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

	// Compute key length
	keyLength := len(accountBytes) - keyVariantLength
	if keyLength <= 0 {
		return nil, fault.ErrInvalidKeyLength
	}

	// return a pointer to the specific account type
	switch keyAlgorithm {
	case ED25519:
		if keyLength != ed25519.PublicKeySize {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := accountBytes[keyVariantLength:]
		account := &Account{
			AccountInterface: &ED25519Account{
				Test:      isTest,
				PublicKey: publicKey,
			},
		}
		return account, nil
	case Nothing:
		if 2 != keyLength {
			return nil, fault.ErrInvalidKeyLength
		}
		publicKey := accountBytes[keyVariantLength:]
		account := &Account{
			AccountInterface: &NothingAccount{
				Test:      isTest,
				PublicKey: publicKey,
			},
		}
		return account, nil
	default:
		return nil, fault.ErrInvalidKeyType
	}
}

func (account *Account) UnmarshalText(s []byte) error {
	a, err := AccountFromBase58(string(s))
	if nil != err {
		return err
	}
	account.AccountInterface = a.AccountInterface
	return nil
}

// ED25519
// -------

// key type code (see enumeration above)
func (account *ED25519Account) KeyType() int {
	return ED25519
}

// fetch the public key as byte slice
func (account *ED25519Account) PublicKeyBytes() []byte {
	return account.PublicKey[:]
}

// check the signature of a message
func (account *ED25519Account) CheckSignature(message []byte, signature Signature) error {

	if ed25519.SignatureSize != len(signature) {
		return fault.ErrInvalidSignature
	}

	if !ed25519.Verify(account.PublicKey[:], message, signature) {
		return fault.ErrInvalidSignature
	}
	return nil
}

// byte slice for encoded key
func (account *ED25519Account) Bytes() []byte {
	keyVariant := byte(ED25519<<algorithmShift) | publicKeyCode
	if account.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, account.PublicKey[:]...)
}

// base58 encoding of encoded key
func (account *ED25519Account) String() string {
	buffer := account.Bytes()
	checksum := sha3.Sum256(buffer)
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// convert an account to its Base58 JSON form
func (account ED25519Account) MarshalText() ([]byte, error) {
	return []byte(account.String()), nil
}

// return whether the public key is in test mode or not
func (account ED25519Account) IsTesting() bool {
	return account.Test
}

// Nothing
// -------

// key type code (see enumeration above)
func (account *NothingAccount) KeyType() int {
	return Nothing
}

// fetch the public key as byte slice
func (account *NothingAccount) PublicKeyBytes() []byte {
	return account.PublicKey[:]
}

// check the signature of a message
func (account *NothingAccount) CheckSignature(message []byte, signature Signature) error {
	return fault.ErrInvalidSignature
}

// byte slice for encoded key
func (account *NothingAccount) Bytes() []byte {
	keyVariant := byte(Nothing<<algorithmShift) | publicKeyCode
	if account.Test {
		keyVariant |= testKeyCode
	}
	return append([]byte{keyVariant}, account.PublicKey[:]...)
}

// base58 encoding of encoded key
func (account *NothingAccount) String() string {
	buffer := account.Bytes()
	checksum := sha3.Sum256(buffer)
	buffer = append(buffer, checksum[:checksumLength]...)
	return util.ToBase58(buffer)
}

// convert an account to its Base58 JSON form
func (account NothingAccount) MarshalText() ([]byte, error) {
	return []byte(account.String()), nil
}

// return whether the public key is in test mode or not
func (account NothingAccount) IsTesting() bool {
	return account.Test
}
