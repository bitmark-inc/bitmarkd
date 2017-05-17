// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/hex"
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"testing"
)

// test encrypt private key and decrypt private key
func TestPrivateKeyEncryptDecrypt(t *testing.T) {
	privateKey1, err := hex.DecodeString("7bc0decf85c70f9612e26682866022243a0a27786a037f762c55e92441bd337568bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Fatalf("decode hex private key failed: %v", err)
	}

	passwords := []string{"test", "123", "444"}

	for _, password := range passwords {
		salt, key, err := hashPassword(password)
		if nil != err {
			t.Fatalf("encryptPassword failed: %v", err)
		}

		encryptPri, err := encryptPrivateKey(privateKey1, key)
		if nil != err {
			t.Fatalf("encryptPrivateKey failed: %v", err)
		}

		key2, err := generateKey(password, salt)
		if nil != err {
			t.Fatalf("generateKey failed: %v", err)
		}

		decryptPri, err := decryptPrivateKey(encryptPri, key2)
		if nil != err {
			t.Fatalf("decryptPrivateKey failed: %v", err)
		}

		if !bytes.Equal(decryptPri, privateKey1) {
			t.Errorf("decrypted privatekey is not equal with original privatekey")
		}
	}
}

func TestPasswordToKey(t *testing.T) {
	publicKey, err := hex.DecodeString("68bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Fatalf("decode hex public key failed: %v", err)
	}
	privateKey, err := hex.DecodeString("7bc0decf85c70f9612e26682866022243a0a27786a037f762c55e92441bd337568bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Fatalf("decode hex private key failed: %v", err)
	}

	passwords := []string{"test", "123", "444"}

	for _, password := range passwords {
		salt, key, err := hashPassword(password)
		if nil != err {
			t.Fatalf("encryptPassword failed: %v", err)
		}
		encryptPri, err := encryptPrivateKey(privateKey, key)
		if nil != err {
			t.Fatalf("encryptPrivateKey failed: %v", err)
		}

		key2, err := generateKey(password, salt)
		if nil != err {
			t.Fatalf("generateKey failed: %v", err)
		}

		decryptPri, err := decryptPrivateKey(encryptPri, key2)
		if nil != err {
			t.Fatalf("decryptPrivateKey failed: %v", err)
		}

		var privateKey2 [64]byte
		copy(privateKey2[:], decryptPri)

		if !checkSignature(publicKey, privateKey) {
			t.Errorf("checkSignature failed")
		}

	}
}

func TestDecryptionToPrivateKey(t *testing.T) {
	publicKey, err := hex.DecodeString("68bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Fatalf("decode hex public key failed: %v", err)
	}
	privateKey, err := hex.DecodeString("7bc0decf85c70f9612e26682866022243a0a27786a037f762c55e92441bd337568bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Fatalf("decode hex private key failed: %v", err)
	}

	control := []struct {
		password   string
		salt       string
		ciphertext string
	}{
		{
			password:   "abcdefghijklmnopqrstuvwxyz",
			salt:       "0477ddc464595a04778be799df57207d",
			ciphertext: "28418cda1e279e79d33f8166078357d840e76e97b2e26f4c1b340d1dcb98a0cf7bd3a4205032ff4f54490fb9cbed756214e14de5a135087b0adb77b502120ed446108603cc0d28dcafea86f6089acf1a",
		},
		{
			password:   "1234567890",
			salt:       "8fc700477e5f4fca3229b12eea9392dd",
			ciphertext: "a8e80fcac3cfd04246baddf9efbd577de0cc732fa56d414c268cdc39e4e364f06e4d2853847e45b44b26b65a38044f8b7a440e37502a26efe0e439230476cbea0ed567baa80a4a6b7a12f392b04b0e84",
		},
		{
			password:   "ephohjie9eewaiRuisiQueeNg9loh0Dee0oorahx7fo2ush7ituaYee2Chu6boeY",
			salt:       "289ff10921138406c0e044460026236a",
			ciphertext: "f1c55988e12a8d26503095f084db15d538046e497e2f019767e8a5ae0a94b114295a3bc0c1dc3151cdcc35819ca28fdc1f40c9aac47e265c57fa9fc46ac48226b08b6a9e4f46ef238531d49c05cf0fb8",
		},
	}

	for i, item := range control {

		var salt configuration.Salt
		err := salt.UnmarshalText([]byte(item.salt))
		if nil != err {
			t.Fatalf("%d: unmarshal salt failed: %v", i, err)
		}

		ciphertext, err := hex.DecodeString(item.ciphertext)
		if nil != err {
			t.Fatalf("%d: decode hex ciphertext failed: %v", i, err)
		}

		key, err := generateKey(item.password, &salt)
		if nil != err {
			t.Errorf("generateKey failed: %v", err)
		}

		// this will get a different ivec each time no will never be the same
		newEncrypted, err := encryptPrivateKey(privateKey, key)
		if nil != err {
			t.Fatalf("%d: encryptPrivateKey failed: %v", i, err)
		}
		// t.Logf("%d: ciphertext: %x", i, newEncrypted) // enable this when creating new items for test

		// make sure encryption does not produce identical results, if it does ivec generation is broken
		if bytes.Equal(ciphertext, newEncrypted) {
			t.Errorf("%d: encryption produced duplicate result - must never happen", i)
			t.Errorf("%d: new ciphertext:    %x", i, newEncrypted)
			t.Errorf("%d: stored ciphertext: %x", i, ciphertext)
		}

		decrypted, err := decryptPrivateKey(ciphertext, key)
		if nil != err {
			t.Fatalf("%d: decryptPrivateKey failed: %v", i, err)
		}

		if !bytes.Equal(decrypted, privateKey) {
			t.Errorf("%d: plaintext actual:   %x", i, decrypted)
			t.Errorf("%d: plaintext expected: %x", i, privateKey)
		}

		var privateKey2 [64]byte
		copy(privateKey2[:], decrypted)

		if !checkSignature(publicKey, privateKey) {
			t.Errorf("%d: checkSignature failed", i)
		}

	}
}
