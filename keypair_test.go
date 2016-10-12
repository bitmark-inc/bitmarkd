// Copyright (c) 2014-2016 Bitmark Inc.
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
		t.Errorf("decode hex private key failed: %v", err)
	}

	passwords := []string{"test", "123", "444"}

	for _, password := range passwords {
		iter, salt, key, err := hashPassword(password)
		if nil != err {
			t.Errorf("encryptPassword failed: %v", err)
		}

		encryptPri, err := encryptPrivateKey(privateKey1, key)
		if nil != err {
			t.Errorf("encryptPrivateKey failed: %v", err)
		}

		key2 := generateKey(password, iter, salt)
		decryptPri, err := decryptPrivateKey(encryptPri, key2)
		if nil != err {
			t.Errorf("decryptPrivateKey failed: %v", err)
		}

		if !bytes.Equal(decryptPri, privateKey1) {
			t.Errorf("decrypted privatekey is not equal with original privatekey")
		}
	}
}

func TestPasswordToKey(t *testing.T) {
	publicKey, err := hex.DecodeString("68bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Errorf("decode hex public key failed: %v", err)
	}
	privateKey, err := hex.DecodeString("7bc0decf85c70f9612e26682866022243a0a27786a037f762c55e92441bd337568bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Errorf("decode hex private key failed: %v", err)
	}

	passwords := []string{"test", "123", "444"}

	for _, password := range passwords {
		iter, salt, key, err := hashPassword(password)
		if nil != err {
			t.Errorf("encryptPassword failed: %v", err)
		}
		encryptPri, err := encryptPrivateKey(privateKey, key)
		if nil != err {
			t.Errorf("encryptPrivateKey failed: %v", err)
		}

		key2 := generateKey(password, iter, salt)
		decryptPri, err := decryptPrivateKey(encryptPri, key2)
		if nil != err {
			t.Errorf("decryptPrivateKey failed: %v", err)
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
		t.Errorf("decode hex public key failed: %v", err)
	}
	privateKey, err := hex.DecodeString("7bc0decf85c70f9612e26682866022243a0a27786a037f762c55e92441bd337568bc0675932a5a83857a71c3670cc2be49c1a7a05408a8dcbb1765728bf69c36")
	if nil != err {
		t.Errorf("decode hex private key failed: %v", err)
	}

	control := []struct {
		password   string
		iter       int
		salt       string
		ciphertext string
	}{
		{
			password:   "abcdefghijklmnopqrstuvwxyz",
			iter:       1234,
			salt:       "0477ddc464595a04778be799df57207d",
			ciphertext: "f7e97756207aa7875a8ee1a0a226a24dc1adc8dcd85b8d9645c14d48dc172d82a7e12c2b41c1748fbc1a7ea7dece5818bf2600b6c0a0f72276068bfe302de537d85e189d0d3caf21a3a7aeef397ac1b5",
		},
		{
			password:   "1234567890",
			iter:       5678,
			salt:       "8fc700477e5f4fca3229b12eea9392dd",
			ciphertext: "d5a99d0bd51d31b3978bb8dbffe947880df94c2b337a11f03af2576ae33c13d52dcd3975704e93af79451a67946186950b9f554a29ad61e11eac5c4454241cf31e3bac795f9acc6867f28888092179dd",
		},
		{
			password:   "ephohjie9eewaiRuisiQueeNg9loh0Dee0oorahx7fo2ush7ituaYee2Chu6boeY",
			iter:       9876,
			salt:       "289ff10921138406c0e044460026236a",
			ciphertext: "3ee336d1c4d1d643b676e084b0173eb0aa94dd71a97b4bbc1738da4b855b2536ca5db68e28551737cf37e1bcdfa2bfb1ef3d3d4862c9607f59f7f3a9d1b28a9d45b53e3c17b5745d902acb962ac5ce14",
		},
	}

	for i, item := range control {

		var iter configuration.Iter
		iter.ConvertIntegerToIter(item.iter)

		var salt configuration.Salt
		err := salt.UnmarshalText([]byte(item.salt))
		if nil != err {
			t.Errorf("%d: unmarshal salt failed: %v", i, err)
		}

		ciphertext, err := hex.DecodeString(item.ciphertext)
		if nil != err {
			t.Errorf("%d: decode hex ciphertext failed: %v", i, err)
		}

		key := generateKey(item.password, &iter, &salt)

		// // this will get a different ivec each time - so just used once to get data above
		// encrypted, err := encryptPrivateKey(privateKey, key)
		// if nil != err {
		// 	t.Errorf("%d: encryptPrivateKey failed: %v", i, err)
		// }
		// t.Logf("%d: ciphertext: %x", i, encrypted)

		decrypted, err := decryptPrivateKey(ciphertext, key)
		if nil != err {
			t.Errorf("%d: decryptPrivateKey failed: %v", i, err)
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
