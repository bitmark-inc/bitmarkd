// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/go-argon2"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"strings"
)

const (
	publicKeySize   = ed25519.PublicKeySize
	privateKeySize  = ed25519.PrivateKeySize
	publicKeyOffset = privateKeySize - publicKeySize
)

var passwordConsole *terminal.Terminal

type KeyPair struct {
	Seed       string
	PublicKey  []byte
	PrivateKey []byte
}

type RawKeyPair struct {
	Seed       string `json:"seed"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func makeRawKeyPair(test bool) (*RawKeyPair, *KeyPair, error) {

	// generate new seed
	seedCore := make([]byte, 32)
	n, err := rand.Read(seedCore)
	if nil != err {
		return nil, nil, err
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

	return makeRawKeyPairFromSeed(seed, test)
}

func makeRawKeyPairFromSeed(seed string, test bool) (*RawKeyPair, *KeyPair, error) {

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

// return of makeKeyPair
type EncryptedKeyPair struct {
	PublicKey           string
	EncryptedPrivateKey string
	EncryptedSeed       string
}

// create a new public/private keypair
// note: private key string must be either:
//       * 64 bytes  =  [32 byte private key][32 byte public key]
//       * 32 bytes  =  [32 byte private key]
//       * "SEED:<base58 encoded seed>"
func makeKeyPair(privateKeyStr string, password string, test bool) (*EncryptedKeyPair, *configuration.PrivateKeyConfig, error) {
	var publicKey, privateKey []byte
	var seed string
	var err error
	// if privateKey is empty, make a new one
	if "" == privateKeyStr {
		raw, pair, err := makeRawKeyPair(test)
		if nil != err {
			return nil, nil, err
		}
		seed = raw.Seed
		publicKey = pair.PublicKey
		privateKey = pair.PrivateKey

	} else if strings.HasPrefix(privateKeyStr, "SEED:") {
		seed = privateKeyStr[5:]
		_, pair, err := makeRawKeyPairFromSeed(seed, test)
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
		if len(privateKey) == privateKeySize {
			publicKey = make([]byte, publicKeySize)
			copy(publicKey, privateKey[publicKeyOffset:])

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

		} else if len(privateKey) == publicKeyOffset {
			// only have the private part, must generate the public part
			b := bytes.NewBuffer(privateKey)
			pub, prv, err := ed25519.GenerateKey(b)
			if nil != err {
				return nil, nil, err
			}
			if !bytes.Equal(privateKey, prv[:publicKeyOffset]) {
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

	privateKeyConfig := &configuration.PrivateKeyConfig{
		Salt: salt.String(),
	}

	return result, privateKeyConfig, nil
}

func accountFromHexPublicKey(publicKey string, test bool) (*account.Account, error) {

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

func hashPassword(password string) (*configuration.Salt, []byte, error) {
	salt, err := configuration.MakeSalt()
	if nil != err {
		return nil, nil, err
	}

	cipher, err := generateKey(password, salt)
	if nil != err {
		return nil, nil, err
	}

	return salt, cipher, nil
}

func generateKey(password string, salt *configuration.Salt) ([]byte, error) {

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

	if len(plaintext) != privateKeySize {
		return nil, fault.ErrKeyLength
	}

	ciphertext := make([]byte, aes.BlockSize+privateKeySize)
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

	if len(ciphertext) != aes.BlockSize+privateKeySize {
		return nil, fault.ErrKeyLength
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
	padding := aes.BlockSize - ((len + countBytes) % aes.BlockSize)

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
	salt, err := configuration.MakeSalt()
	if nil != err {
		return false
	}
	message := salt.String() + "Bitmark Command Line Interface"
	signature := ed25519.Sign(privateKey, []byte(message))
	return ed25519.Verify(publicKey, []byte(message), signature)
}

func getTerminal() (*terminal.Terminal, int, *terminal.State) {
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}

	if nil != passwordConsole {
		return passwordConsole, 0, oldState
	}

	tmpIO, err := os.OpenFile("/dev/tty", os.O_RDWR, os.ModePerm)
	if nil != err {
		panic("No console")
	}

	passwordConsole = terminal.NewTerminal(tmpIO, "bitmark-cli: ")

	return passwordConsole, 0, oldState
}

func promptPasswordReader() (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("Set identity password(length >= 8): ")
	if nil != err {
		fmt.Printf("Get password fail: %s\n", err)
		return "", err
	}
	terminal.Restore(fd, state)

	passwordLen := len(password)
	if passwordLen < 8 {
		return "", fault.ErrPasswordLength
	}

	console, fd, state = getTerminal()
	verifyPassword, err := console.ReadPassword("Verify password: ")
	if nil != err {
		fmt.Printf("verify failed: %s\n", err)
		return "", fault.ErrVerifiedPassword
	}
	terminal.Restore(fd, state)

	if password != verifyPassword {
		return "", fault.ErrVerifiedPassword
	}

	return password, nil
}

func promptCheckPasswordReader() (string, error) {
	console, fd, state := getTerminal()
	password, err := console.ReadPassword("password: ")
	if nil != err {
		fmt.Printf("Get password fail: %s\n", err)
		return "", err
	}
	terminal.Restore(fd, state)

	return password, nil
}

func promptAndCheckPassword(issuer *configuration.IdentityType) (*KeyPair, error) {
	password, err := promptCheckPasswordReader()
	if nil != err {
		return nil, err
	}

	keyPair, err := verifyPassword(password, issuer)
	if nil != err {
		return nil, err
	}
	return keyPair, nil
}

func verifyPassword(password string, identity *configuration.IdentityType) (*KeyPair, error) {
	salt := new(configuration.Salt)
	salt.UnmarshalText([]byte(identity.Private_key_config.Salt))

	key, err := generateKey(password, salt)
	if nil != err {
		return nil, err
	}

	ciphertext, err := hex.DecodeString(identity.Private_key)
	if nil != err {
		return nil, err
	}

	privateKey, err := decryptPrivateKey(ciphertext, key)
	if nil != err {
		return nil, err
	}

	publicKey, err := hex.DecodeString(identity.Public_key)
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

	keyPair := KeyPair{
		Seed:       seed,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	return &keyPair, nil
}

func publicKeyFromIdentity(name string, identities []configuration.IdentityType) (*KeyPair, error) {

	for _, identity := range identities {
		if name != identity.Name {
			continue
		}
		publicKey, err := hex.DecodeString(identity.Public_key)
		if nil != err {
			return nil, err
		}

		keyPair := KeyPair{
			PublicKey: publicKey,
		}
		return &keyPair, nil
	}
	return nil, fault.ErrNotFoundIdentity
}
