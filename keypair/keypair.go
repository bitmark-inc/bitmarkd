// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package keypair

import (
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
)

var (
	ErrKeyLength        = fault.InvalidError("key length is invalid")
	ErrNotFoundIdentity = fault.NotFoundError("identity name not found")
)

// KeyPair - structure to hold public and private keys and the seed
// that was used to generate them
type KeyPair struct {
	Seed       string
	PublicKey  []byte
	PrivateKey []byte
}

// RawKeyPair - text version of seed and keys
type RawKeyPair struct {
	Seed       string `json:"seed"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// NewSeed - create a new seed from secure random data
func NewSeed(test bool) (string, error) {
	// generate new seed
	seedCore := make([]byte, 32)
	n, err := rand.Read(seedCore)
	if nil != err {
		return "", err
	}
	if 32 != n {
		panic("too few random bytes")
	}
	net := 0x00
	if test {
		net = 0x01
	}
	packedSeed := []byte{0x5a, 0xfe, 0x01, byte(net)}
	packedSeed = append(packedSeed, seedCore...)
	checksum := sha3.Sum256(packedSeed)
	packedSeed = append(packedSeed, checksum[:4]...)

	seed := util.ToBase58(packedSeed)
	return seed, nil
}

// MakeRawKeyPair - create new seed and generate public/private keys from it
func MakeRawKeyPair(test bool) (*RawKeyPair, *KeyPair, error) {
	seed, err := NewSeed(test)
	if err != nil {
		return nil, nil, err
	}
	return MakeRawKeyPairFromSeed(seed, test)
}

// MakeRawKeyPairFromSeed - generate public/private keys from existing seed
func MakeRawKeyPairFromSeed(seed string, test bool) (*RawKeyPair, *KeyPair, error) {

	privateKey, err := account.PrivateKeyFromBase58Seed(seed)
	if nil != err {
		return nil, nil, err
	}

	keyPair := KeyPair{
		Seed:       seed,
		PublicKey:  privateKey.Account().PublicKeyBytes(),
		PrivateKey: privateKey.PrivateKeyBytes(),
	}

	rawKeyPair := RawKeyPair{
		Seed:       seed,
		PublicKey:  hex.EncodeToString(privateKey.Account().PublicKeyBytes()),
		PrivateKey: hex.EncodeToString(privateKey.PrivateKeyBytes()),
	}

	return &rawKeyPair, &keyPair, nil
}

// AccountFromHexPublicKey - create an account from a hexadecimal public key
func AccountFromHexPublicKey(publicKey string, test bool) (*account.Account, error) {

	k, err := hex.DecodeString(publicKey)
	if nil != err {
		return nil, err
	}

	account := &account.Account{
		AccountInterface: &account.ED25519Account{
			Test:      test,
			PublicKey: k,
		},
	}
	return account, nil
}
