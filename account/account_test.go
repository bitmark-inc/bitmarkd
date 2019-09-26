// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package account_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// Test account functionality

type accountTest struct {
	algorithm     int
	testnet       bool
	zero          bool
	publicKey     []byte
	base58Account string
}

// Valid account
var testAccount = []accountTest{
	{
		algorithm:     account.ED25519,
		testnet:       false,
		zero:          false,
		publicKey:     decodeHex("60b3c6e20cfff7091a86488b1656b96ec0a2f69907e2c035175918f42c37d72e"),
		base58Account: "anF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXaktvLCj",
	},
	{
		algorithm:     account.ED25519,
		testnet:       true,
		zero:          false,
		publicKey:     decodeHex("731114267f15754a5fce4aaed8380b28aff25af7b378b011d92ef7b3f08910db"),
		base58Account: "eopaSeB7uiSVMdAmTrijq3W2MCWA5KHZrZvm5QLFGRVd3oWNe2",
	},
	{
		algorithm:     account.ED25519,
		testnet:       true,
		zero:          false,
		publicKey:     decodeHex("cb6ff605f79deba3deb0c5122e40359a258481c151dffc176a2da5e8bc87cd2e"),
		base58Account: "fUjtNvmUJn7yJ7PVP7NT2FZbKDrudFxLVBHkwLJFgKWmGsPNVi",
	},
	{
		algorithm:     account.ED25519,
		testnet:       true,
		zero:          true,
		publicKey:     decodeHex("0000000000000000000000000000000000000000000000000000000000000000"),
		base58Account: "dw9MQXcC5rJZb3QE1nz86PiQAheMP1dx9M3dr52tT8NNs14m33",
	},
	{
		algorithm:     account.ED25519,
		testnet:       false,
		zero:          true,
		publicKey:     decodeHex("0000000000000000000000000000000000000000000000000000000000000000"),
		base58Account: "a3ezwdYVEVrHwszQrYzDTCAZwUD3yKtNsCq9YhEu97bPaGAKy1",
	},
	{
		algorithm:     account.Nothing,
		testnet:       false,
		zero:          false,
		publicKey:     decodeHex("12fa"),
		base58Account: "3MvykBZzN",
	},
	{
		algorithm:     account.Nothing,
		testnet:       false,
		zero:          true,
		publicKey:     decodeHex("0000"),
		base58Account: "3CUwbPENE",
	},
}

type invalid struct {
	str string
	err error
}

// Invalid account
var testInvalidAccountFromBase58 = []invalid{
	{"3gLJjLSociTmf4kgL3ztUK;tgADFvg9yjXt1jFbEx9KgpEEAFn", fault.CannotDecodeAccount}, // invalid base58 string
	{"anF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXaktvLDj", fault.ChecksumMismatch},    // checksum mismatch
	{"WjbRFkA9dhmMKnKTuufZ1sVD4E4H1NRnsmwjMKNHHRSCvDm5bXPV", fault.InvalidKeyType},    // undefined key algorithm
	{"YqVxD4vazrrnxnLH2MzCHJedPPz1VKHnKbVfya39nF96ABAYes", fault.NotPublicKey},        // private key
	{"anF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXaktvLC", fault.NotPublicKey},         // truncated
	{"nF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXaktvLCj", fault.NotPublicKey},         // truncated
}

// show manually created accounts
// this has to be changed if account.go is modified
// it is used to print the base58Account for testAccount above
func TestValid(t *testing.T) {

loop:
	for index, test := range testAccount {
		testnet := 0x00
		if test.testnet {
			testnet = 0x02
		}

		buffer := []byte{byte(test.algorithm<<4 | 0x01 | testnet)}
		buffer = append(buffer, test.publicKey...)
		account, err := account.AccountFromBytes(buffer)
		if nil != err {
			t.Errorf("%d: Create account from bytes failed: %s", index, err)
			continue loop
		}
		t.Logf("%d: result: %s", index, account)
		t.Logf("%d:    hex: %x", index, account.Bytes())

		if !bytes.Equal(buffer, account.Bytes()) {
			t.Errorf("%d: account bytes: %x does not match: %x", index, account.Bytes(), buffer)
		}

		accountIsZero := true
	check_for_zero:
		for _, b := range account.PublicKeyBytes() {
			if 0 != b {
				accountIsZero = false
				break check_for_zero
			}
		}
		if test.zero {
			if !accountIsZero {
				t.Errorf("%d: account bytes: %x not zero, but should be zero", index, account.PublicKeyBytes())
			}
			if !account.IsZero() {
				t.Errorf("%d: account.IsZero() incorrectly returned false", index)
			}
		} else {
			if accountIsZero {
				t.Errorf("%d: account bytes: %x are all zero, but should not be", index, account.PublicKeyBytes())
			}
			if account.IsZero() {
				t.Errorf("%d: account.IsZero() incorrectly returned true", index)
			}
		}
	}
}

// From valid base58 string to account
func TestValidBase58(t *testing.T) {
loop:
	for index, test := range testAccount {
		acc, err := account.AccountFromBase58(test.base58Account)
		if nil != err {
			t.Errorf("%d: from base58 error: %s", index, err)
			continue loop
		}
		if acc.IsTesting() != test.testnet {
			t.Errorf("%d: from base58 testnet: %t  expected: %t", index, acc.IsTesting(), test.testnet)
		}
		if acc.KeyType() != test.algorithm {
			t.Errorf("%d: from base58 type: %d  expected: %d", index, acc.KeyType(), test.algorithm)
		}
		if !bytes.Equal(acc.PublicKeyBytes(), test.publicKey) {
			t.Errorf("%d: from base58 pubkey: %x  expected %x", index, acc.PublicKeyBytes(), test.publicKey)
		}
		if acc.String() != test.base58Account {
			t.Errorf("%d: to base58: got: %s  expected %s", index, acc, test.base58Account)
		}

		// test unmarshal JSON
		j := `"` + test.base58Account + `"`
		var a account.Account
		err = json.Unmarshal([]byte(j), &a)
		if nil != err {
			t.Errorf("%d: from JSON string error: %s", index, err)
			continue loop
		}
		t.Logf("%d: from JSON: %#v", index, a)

		buffer, _ := json.Marshal(a)
		t.Logf("%d: account to JSON: %s", index, buffer)
		if j != string(buffer) {
			t.Errorf("%d: marshal JSON:failed: expected %s  actual: %s", index, j, buffer)
		}

	}
}

// Test invalid account parsing
// From account base58 encoded to account
func TestInvalidBase58(t *testing.T) {
	for index, test := range testInvalidAccountFromBase58 {
		_, err := account.AccountFromBase58(test.str)
		if test.err != err {
			t.Errorf("invalid base58 string: %d failed: expected: %q actual: %q", index, test.err, err)
		}
	}
}

// Decode the hex string and return []byte.
//
// This is only used in the tests as the source is pre-prepared, so that there won't be any error
func decodeHex(hexStr string) []byte {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		panic(err)
	}
	return b
}
