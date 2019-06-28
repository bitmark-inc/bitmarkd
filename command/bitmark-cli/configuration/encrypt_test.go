// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package encrypt

import (
	"testing"
)

// test encrypt and decrypt one string with various passwords
func TestEncryptDecrypt(t *testing.T) {

	plainText := "The Quick Brown Fox Jumps Over The Lazy Dog"

	passwords := []string{"test", "123", "444", "m,erRGhtk%$33ug62sd al/fajfb.adv"}

	for _, password := range passwords {
		salt, key, err := hashPassword(password)
		if nil != err {
			t.Fatalf("hash error: %s", err)
		}

		encrypted, err := encryptData(plainText, key)
		if nil != err {
			t.Fatalf("encrypt error: %s", err)
		}

		key2, err := generateKey(password, salt)
		if nil != err {
			t.Fatalf("generateKey error: %s", err)
		}

		decrypted, err := decryptData(encrypted, key2)
		if nil != err {
			t.Fatalf("decrypt error: %s", err)
		}

		if decrypted != plainText {
			t.Errorf("decrypt: expected: %s", decrypted)
			t.Errorf("decrypt: actual:   %s", plainText)
		}
	}
}

func TestDecryptionAnNoDuplication(t *testing.T) {

	plainText := "This is some text for testing 1234567890"

	control := []struct {
		password   string
		salt       string
		ciphertext string
	}{
		{
			password:   "abcdefghijklmnopqrstuvwxyz",
			salt:       "bcd97512f9994ac83a6c63ea4302d45484f1d66fcf65067e3c93f30335c9439c",
			ciphertext: "5f86e901239b68fae1940b1879f930c4944c587e70abdaf6855c9848205ba713eff0b8d69339e4e056ddadf8801ea5817c554cd630ce5fe813c9b61ddc8e7a0f03bb51248568e5188df0fee891101e12",
		},
		{
			password:   "1234567890",
			salt:       "8fc700477e5f4fca3229b12eea9392dd0477ddc464595a04778be799df57207d",
			ciphertext: "411cf961447161105e29c9e4561553d68832f5d8b848a577c1624649fef7732a436634ba7e31619f15a63ee711e40e0ab4b5068f2114ec79e12b88c53c3e4a2a4c2fe620301cb573f009fccf5191931b",
		},
		{
			password:   "ephohjie9eewaiRuisiQueeNg9loh0Dee0oorahx7fo2ush7ituaYee2Chu6boeY",
			salt:       "f30834ef0b39cb38c3d4b668e84adfa95d4c5b7f4e812c5579fa76ee5ca09b18",
			ciphertext: "06605847417fc548816a654f6766baafcebc192750736d0d750ad873649f7e7a840e7d04e83efbb1f4c15a23ef8dc21a2930556cfdaac72c76d24629c97eb1f671df8ebbf8c492940e35b6141ce6c0c1",
		},
	}

	for i, item := range control {

		var salt Salt
		err := salt.UnmarshalText([]byte(item.salt))
		if nil != err {
			t.Fatalf("%d: unmarshal salt error: %s", i, err)
		}

		key, err := generateKey(item.password, &salt)
		if nil != err {
			t.Fatalf("generateKey failed: %s", err)
		}

		// this will get a different ivec each time no will never be the same
		newEncrypted, err := encryptData(plainText, key)
		if nil != err {
			t.Fatalf("%d: encrypt error: %s", i, err)
		}

		//t.Logf("%d: ciphertext: %s", i, newEncrypted) // enable this when creating new items for test

		// make sure encryption does not produce identical results, if it does ivec generation is broken
		if item.ciphertext == newEncrypted {
			t.Errorf("%d: encryption produced duplicate result - must never happen", i)
			t.Errorf("%d: new ciphertext:    %s", i, newEncrypted)
			t.Errorf("%d: stored ciphertext: %s", i, item.ciphertext)
		}

		decrypted, err := decryptData(item.ciphertext, key)
		if nil != err {
			t.Fatalf("%d: decrypt error: %s", i, err)
		}

		if decrypted != plainText {
			t.Errorf("%d: plaintext actual:   %s", i, decrypted)
			t.Errorf("%d: plaintext expected: %s", i, plainText)
		}

		// test password mismatch
		key, err = generateKey("A Bad Password", &salt)
		if nil != err {
			t.Fatalf("generateKey failed: %s", err)
		}

		_, err = decryptData(item.ciphertext, key)
		if nil == err {
			t.Errorf("%d: unexpected decryption success", i)
		}
	}
}
