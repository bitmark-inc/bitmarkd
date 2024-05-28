// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package configuration

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/go-argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

type Private struct {
	PrivateKey  *account.PrivateKey `json:"privateKey"`
	Seed        string              `json:"seed"`
	Description string              `json:"description"`
}

// decryptIdentity - check if password unlocks data in the configuration file
func decryptIdentity(password string, identity *Identity) (*Private, error) {

	salt := new(Salt)
	err := salt.UnmarshalText([]byte(identity.Salt))
	if err != nil || identity.Data == "" {
		return nil, fault.NotPrivateKey
	}

	key, err := generateKey(password, salt)
	if err != nil {
		return nil, err
	}

	seed, err := decryptData(identity.Data, key)
	if err != nil {
		return nil, fault.WrongPassword
	}

	privateKey, err := account.PrivateKeyFromBase58Seed(seed)
	if err != nil {
		return nil, err
	}

	r := Private{
		PrivateKey:  privateKey,
		Seed:        seed,
		Description: identity.Description,
	}
	return &r, nil
}

func hashPassword(password string) (*Salt, *[32]byte, error) {
	salt, err := MakeSalt()
	if err != nil {
		return nil, nil, err
	}

	cipher, err := generateKey(password, salt)
	if err != nil {
		return nil, nil, err
	}

	return salt, cipher, nil
}

func generateKey(password string, salt *Salt) (*[32]byte, error) {

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
	if err != nil {
		return nil, err
	}

	var secretKey [32]byte
	copy(secretKey[:], hash)

	return &secretKey, nil
}

// encrypt a string and convert to hex
func encryptData(data string, secretKey *[32]byte) (string, error) {

	// ensure data not too small or too large
	l := len(data)
	if l < 32 || l >= 16384 {
		return "", fault.CryptoFailed
	}

	// must use a different nonce for each message you encrypt with the
	// same key. Since the nonce here is 192 bits long, a random value
	// provides a sufficiently small probability of repeats.
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fault.CryptoFailed
	}

	// encrypt
	ciphertext := secretbox.Seal(nonce[:], []byte(data), &nonce, secretKey)

	// return as hex string
	return hex.EncodeToString(ciphertext), nil
}

// decrypt a hex string and return plaintext
func decryptData(ciphertext string, secretKey *[32]byte) (string, error) {

	if ciphertext == "" {
		return "", fault.CryptoFailed
	}

	encrypted, err := hex.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	if len(encrypted) <= 24 {
		return "", fault.CryptoFailed
	}

	// When you decrypt, you must use the same nonce and key you used to
	// encrypt the message. A way to achieve this is to store the nonce
	// alongside the encrypted message
	var nonce [24]byte
	copy(nonce[:], encrypted[:24])

	decrypted, ok := secretbox.Open(nil, encrypted[24:], &nonce, secretKey)
	if !ok {
		return "", fault.CryptoFailed
	}

	return string(decrypted), nil
}
