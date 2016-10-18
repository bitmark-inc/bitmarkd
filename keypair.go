// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
)

const (
	iterBaseRange  = 5000
	publicKeySize  = ed25519.PublicKeySize
	privateKeySize = ed25519.PrivateKeySize
)

var passwordConsole *terminal.Terminal

type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

type RawKeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func makeRawKeyPair() error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if nil != err {
		return err
	}

	rawKeyPair := RawKeyPair{
		PublicKey:  hex.EncodeToString(publicKey),
		PrivateKey: hex.EncodeToString(privateKey),
	}
	if b, err := json.MarshalIndent(rawKeyPair, "", "  "); nil != err {
		return err
	} else {
		fmt.Printf("%s\n", b)
	}
	return nil
}

// create a new public/private keypair
func makeKeyPair(privateKeyStr string, password string) (string, string, *configuration.PrivateKeyConfig, error) {
	var publicKey, privateKey []byte
	var err error
	// if privateKey is empty, make a new one
	if "" == privateKeyStr {
		publicKey, privateKey, err = ed25519.GenerateKey(rand.Reader)
		if nil != err {
			return "", "", nil, err
		}
	} else {
		privateKey, err = hex.DecodeString(privateKeyStr)
		if nil != err {
			return "", "", nil, err
		}
		//check privateKey is valid
		if len(privateKey) != privateKeySize {
			return "", "", nil, fault.ErrInvalidPrivateKey
		}

		publicKey = make([]byte, publicKeySize)
		copy(publicKey, privateKey[32:])
	}

	iter, salt, key, err := hashPassword(password)
	if nil != err {
		return "", "", nil, err
	}

	// use key to encrypt private key
	encryptedPrivateKey, err := encryptPrivateKey(privateKey, key)
	if nil != err {
		return "", "", nil, err
	}

	publicStr := hex.EncodeToString(publicKey)
	privateStr := hex.EncodeToString(encryptedPrivateKey)
	privateKeyConfig := &configuration.PrivateKeyConfig{
		Iter: iter.Integer(),
		Salt: salt.String(),
	}

	return publicStr, privateStr, privateKeyConfig, nil
}

func hashPassword(password string) (*configuration.Iter, *configuration.Salt, []byte, error) {
	salt, err := configuration.MakeSalt()
	if nil != err {
		return nil, nil, nil, err
	}
	iter, err := configuration.MakeIter()
	if nil != err {
		return nil, nil, nil, err
	}

	cipher := generateKey(password, iter, salt)

	return iter, salt, cipher, nil
}

func generateKey(password string, iter *configuration.Iter, salt *configuration.Salt) []byte {
	saltBytes := salt.Bytes()
	iterInt := iter.Integer()
	return pbkdf2.Key([]byte(password), saltBytes, iterInt, 32, sha512.New)
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
	password, err := console.ReadPassword("Set identity password( >= 8): ")
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
	iter := new(configuration.Iter)
	salt := new(configuration.Salt)
	iter.ConvertIntegerToIter(identity.Private_key_config.Iter)
	salt.UnmarshalText([]byte(identity.Private_key_config.Salt))

	key := generateKey(password, iter, salt)

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

	keyPair := KeyPair{
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
