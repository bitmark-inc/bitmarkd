// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"
	"strings"

	"golang.org/x/crypto/ed25519"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/bitmark-inc/go-argon2"
)

const (
	PublicKeySize   = ed25519.PublicKeySize
	PrivateKeySize  = ed25519.PrivateKeySize
	PublicKeyOffset = PrivateKeySize - PublicKeySize
)

// IdentityType - full access to data (includes private data)
type IdentityType struct {
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	PublicKey        string         `json:"public_key"`
	PrivateKey       string         `json:"private_key"`
	Seed             string         `json:"seed"`
	PrivateKeyConfig PrivateKeySalt `json:"private_key_config"`
}

// PrivateKeySalt - structure to hols per account salt
type PrivateKeySalt struct {
	Salt string `json:"salt"`
}

// EncryptedKeyPair - return of makeKeyPair
type EncryptedKeyPair struct {
	PublicKey           string
	EncryptedPrivateKey string
	EncryptedSeed       string
}

// MakeKeyPair - create a new public/private keypair
// note: private key string must be either:
//       * 64 bytes  =  [32 byte private key][32 byte public key]
//       * 32 bytes  =  [32 byte private key]
//       * "SEED:<base58 encoded seed>"
func MakeKeyPair(privateKeyStr string, password string, test bool) (*EncryptedKeyPair, *PrivateKeySalt, error) {
	var publicKey, privateKey []byte
	var seed string
	var err error
	// if privateKey is empty, make a new one
	if "" == privateKeyStr {
		raw, pair, err := keypair.MakeRawKeyPair(test)
		if nil != err {
			return nil, nil, err
		}
		seed = raw.Seed
		publicKey = pair.PublicKey
		privateKey = pair.PrivateKey

	} else if strings.HasPrefix(privateKeyStr, "SEED:") {
		seed = privateKeyStr[5:]
		_, pair, err := keypair.MakeRawKeyPairFromSeed(seed, test)
		if nil != err {
			return nil, nil, err
		}
		publicKey = pair.PublicKey
		privateKey = pair.PrivateKey
	} else {
		// DEBUG: fmt.Printf("Hex key: %s\n", privateKeyStr)
		privateKey, err = hex.DecodeString(privateKeyStr)
		if nil != err {
			return nil, nil, err
		}
		// check privateKey is valid
		if len(privateKey) == PrivateKeySize {
			publicKey = make([]byte, PublicKeySize)
			copy(publicKey, privateKey[PublicKeyOffset:])

			b := bytes.NewBuffer(privateKey)
			pub, prv, err := ed25519.GenerateKey(b)
			if nil != err {
				return nil, nil, err
			}
			if !bytes.Equal(privateKey, prv) {
				return nil, nil, fault.ErrUnableToRegenerateKeys
			}
			if !bytes.Equal(publicKey, pub) {
				return nil, nil, fault.ErrUnableToRegenerateKeys
			}

		} else if len(privateKey) == PublicKeyOffset {
			// only have the private part, must generate the public part
			b := bytes.NewBuffer(privateKey)
			pub, prv, err := ed25519.GenerateKey(b)
			if nil != err {
				return nil, nil, err
			}
			if !bytes.Equal(privateKey, prv[:PublicKeyOffset]) {
				return nil, nil, fault.ErrUnableToRegenerateKeys
			}
			privateKey = prv
			publicKey = pub
		} else {
			return nil, nil, fault.ErrInvalidPrivateKey
		}

	}
	// DEBUG: fmt.Printf("seed: %q\n", seed)
	// DEBUG: fmt.Printf("privateKey: %x\n", privateKey)
	// DEBUG: fmt.Printf("publicKey: %x\n", publicKey)

	salt, key, err := hashPassword(password)
	if nil != err {
		return nil, nil, err
	}

	// use key to encrypt private key
	encryptedPrivateKey, err := encryptPrivateKey(privateKey, key)
	if nil != err {
		return nil, nil, err
	}

	// use key to encrypt seed
	encryptedSeed, err := encryptSeed(seed, key)
	if nil != err {
		return nil, nil, err
	}

	result := &EncryptedKeyPair{
		PublicKey:           hex.EncodeToString(publicKey),
		EncryptedPrivateKey: hex.EncodeToString(encryptedPrivateKey),
		EncryptedSeed:       hex.EncodeToString(encryptedSeed),
	}

	privateKeyConfig := &PrivateKeySalt{
		Salt: salt.String(),
	}

	return result, privateKeyConfig, nil
}

func hashPassword(password string) (*Salt, []byte, error) {
	salt, err := MakeSalt()
	if nil != err {
		return nil, nil, err
	}

	cipher, err := generateKey(password, salt)
	if nil != err {
		return nil, nil, err
	}

	return salt, cipher, nil
}

func generateKey(password string, salt *Salt) ([]byte, error) {

	saltBytes := salt.Bytes()

	ctx := &argon2.Context{
		Iterations:  5,
		Memory:      1 << 16,
		Parallelism: 4,
		HashLen:     32,
		Mode:        argon2.ModeArgon2i,
		Version:     argon2.Version13,
	}

	hash, err := argon2.Hash(ctx, []byte(password), saltBytes)
	return hash, err
}

// Encrypt private key
func encryptPrivateKey(plaintext []byte, key []byte) ([]byte, error) {
	block, error := aes.NewCipher(key)
	if nil != error {
		return nil, error
	}

	if len(plaintext) != PrivateKeySize {
		return nil, fault.ErrInvalidKeyLength
	}

	ciphertext := make([]byte, aes.BlockSize+PrivateKeySize)
	iv := ciphertext[:aes.BlockSize]
	if _, error = io.ReadFull(rand.Reader, iv); nil != error {
		return nil, error
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

func decryptPrivateKey(ciphertext []byte, key []byte) ([]byte, error) {
	block, error := aes.NewCipher(key)
	if nil != error {
		return nil, error
	}

	if len(ciphertext) != aes.BlockSize+PrivateKeySize {
		return nil, fault.ErrInvalidKeyLength
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	return ciphertext, nil
}

func encryptSeed(seed string, key []byte) ([]byte, error) {
	block, error := aes.NewCipher(key)
	if nil != error {
		return nil, error
	}
	len := len(seed)
	if len > 1024 {
		panic("encrypting seed > 1024 bytes")
	}

	const countBytes = 2
	padding := aes.BlockSize - (len+countBytes)%aes.BlockSize

	plaintext := make([]byte, len+countBytes+padding)
	plaintext[0] = byte(len / 256)
	plaintext[1] = byte(len % 256)
	copy(plaintext[2:], []byte(seed))

	ciphertext := make([]byte, aes.BlockSize+len+countBytes+padding)
	iv := ciphertext[:aes.BlockSize]
	if _, error = io.ReadFull(rand.Reader, iv); nil != error {
		return nil, error
	}
	mode := cipher.NewCBCEncrypter(block, iv)

	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

func decryptSeed(ciphertext []byte, key []byte) (string, error) {

	if nil == ciphertext || 0 == len(ciphertext) {
		return "", nil
	}
	block, error := aes.NewCipher(key)
	if nil != error {
		return "", error
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	len := int(ciphertext[0])<<8 + int(ciphertext[1])

	return string(ciphertext[2 : len+2]), nil
}

func checkSignature(publicKey []byte, privateKey []byte) bool {
	salt, err := MakeSalt()
	if nil != err {
		return false
	}
	message := salt.String() + "Bitmark Command Line Interface"
	signature := ed25519.Sign(privateKey, []byte(message))
	return ed25519.Verify(publicKey, []byte(message), signature)
}

// VerifyPassword - check if password unlocks data in the configuration file
func VerifyPassword(password string, identity *IdentityType) (*keypair.KeyPair, error) {
	salt := new(Salt)
	salt.UnmarshalText([]byte(identity.PrivateKeyConfig.Salt))

	key, err := generateKey(password, salt)
	if nil != err {
		return nil, err
	}

	ciphertext, err := hex.DecodeString(identity.PrivateKey)
	if nil != err {
		return nil, err
	}

	privateKey, err := decryptPrivateKey(ciphertext, key)
	if nil != err {
		return nil, err
	}

	publicKey, err := hex.DecodeString(identity.PublicKey)
	if nil != err {
		return nil, err
	}

	if !checkSignature(publicKey, privateKey) {
		return nil, fault.ErrWrongPassword
	}

	ciphertext, err = hex.DecodeString(identity.Seed)
	if nil != err {
		return nil, err
	}
	seed, err := decryptSeed(ciphertext, key)
	if nil != err {
		return nil, err
	}

	keyPair := keypair.KeyPair{
		Seed:       seed,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	return &keyPair, nil
}

// PublicKeyFromString - convert an Identity, Base58 or Hex string string to a public key byte slice
func PublicKeyFromString(name string, identities []IdentityType, testnet bool) (*keypair.KeyPair, error) {

loop:
	for _, identity := range identities {
		if name != identity.Name {
			continue loop
		}
		publicKey, err := hex.DecodeString(identity.PublicKey)
		if nil != err {
			return nil, err
		}

		keyPair := keypair.KeyPair{
			PublicKey: publicKey,
		}
		return &keyPair, nil
	}

	// try name as a base58 value
	acc, err := account.AccountFromBase58(name)
	if nil == err {
		if acc.IsTesting() != testnet {
			return nil, fault.ErrWrongNetworkForPublicKey
		}
		keyPair := keypair.KeyPair{
			PublicKey: acc.PublicKeyBytes(),
		}
		return &keyPair, nil
	}

	// try if it is a hex public key
	newPublicKey, err := hex.DecodeString(name)
	if nil == err && len(newPublicKey) == PublicKeySize {
		keyPair := keypair.KeyPair{
			PublicKey: newPublicKey,
		}
		return &keyPair, nil
	}

	return nil, fault.ErrIdentityNameNotFound
}
