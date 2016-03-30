// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"testing"
	// "fmt"
)

// test encrypt private key and decrypt private key
func Test(t *testing.T) {
	// publickey1 := []byte{
	// 		104, 188, 6, 117, 147, 42, 90, 131,
	// 		133, 122, 113, 195, 103, 12, 194, 190,
	// 		73, 193, 167, 160, 84, 8, 168, 220,
	// 		187, 23, 101, 114, 139, 246, 156, 54,
	// }
	privatekey1 := []byte{
		123, 192, 222, 207, 133, 199, 15, 150,
		18, 226, 102, 130, 134, 96, 34, 36,
		58, 10, 39, 120, 106, 3, 127, 118,
		44, 85, 233, 36, 65, 189, 51, 117,
		104, 188, 6, 117, 147, 42, 90, 131,
		133, 122, 113, 195, 103, 12, 194, 190,
		73, 193, 167, 160, 84, 8, 168, 220,
		187, 23, 101, 114, 139, 246, 156, 54,
	}

	passwords := []string{"test", "123", "444"}

	for _, password := range passwords {
		iter, salt, encryptPsd, err := encryptPassword(password)
		if nil != err {
			t.Errorf("encryptPassword failed: %v", err)
		}

		encryptPri, err := encryptPrivateKey(privatekey1, encryptPsd)
		if nil != err {
			t.Errorf("encryptPrivateKey failed: %v", err)
		}

		encryptPsd2 := generateKey(password, iter, salt)
		decryptPri, err := decryptPrivateKey(encryptPri, encryptPsd2)
		if nil != err {
			t.Errorf("decryptPrivateKey failed: %v", err)
		}

		if !bytes.Equal(decryptPri, privatekey1) {
			t.Errorf("decrypted privatekey is not qual with original privatekey")
		}
	}
}

func Test2(t *testing.T) {
	publickey1 := [32]byte{
		104, 188, 6, 117, 147, 42, 90, 131,
		133, 122, 113, 195, 103, 12, 194, 190,
		73, 193, 167, 160, 84, 8, 168, 220,
		187, 23, 101, 114, 139, 246, 156, 54,
	}
	privatekey1 := []byte{
		123, 192, 222, 207, 133, 199, 15, 150,
		18, 226, 102, 130, 134, 96, 34, 36,
		58, 10, 39, 120, 106, 3, 127, 118,
		44, 85, 233, 36, 65, 189, 51, 117,
		104, 188, 6, 117, 147, 42, 90, 131,
		133, 122, 113, 195, 103, 12, 194, 190,
		73, 193, 167, 160, 84, 8, 168, 220,
		187, 23, 101, 114, 139, 246, 156, 54,
	}

	passwords := []string{"test", "123", "444"}

	for _, password := range passwords {
		iter, salt, encryptPsd, err := encryptPassword(password)
		if nil != err {
			t.Errorf("encryptPassword failed: %v", err)
		}
		encryptPri, err := encryptPrivateKey(privatekey1, encryptPsd)
		if nil != err {
			t.Errorf("encryptPrivateKey failed: %v", err)
		}

		encryptPsd2 := generateKey(password, iter, salt)
		decryptPri, err := decryptPrivateKey(encryptPri, encryptPsd2)
		if nil != err {
			t.Errorf("decryptPrivateKey failed: %v", err)
		}

		var tmpPri [64]byte
		for i := 0; i < 64; i++ {
			tmpPri[i] = decryptPri[i]
		}

		var publicKey *[32]byte
		var privateKey *[64]byte
		publicKey = &publickey1
		privateKey = &tmpPri

		if !checkSignature(password, publicKey, privateKey) {
			t.Errorf("checkSignature failed")
		}

	}
}
