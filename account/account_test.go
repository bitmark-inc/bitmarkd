// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package account_test

import (
	"bytes"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
	"testing"
)

// Test account functionality

type accountTest struct {
	algorithm     int
	publicKey     []byte
	base58Account string
}

// Valid account
var testAccount = []accountTest{
	{account.ED25519, decodeHex("60b3c6e20cfff7091a86488b1656b96ec0a2f69907e2c035175918f42c37d72e"), "anF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXaktvLCj"},
	{account.Nothing, decodeHex("12fa"), "3MvykBZzN"},
}

type invalid struct {
	str string
	err error
}

// Invalid account
var testInvalidAccountFromBase58 = []invalid{
	{"3gLJjLSociTmf4kgL3ztUK;tgADFvg9yjXt1jFbEx9KgpEEAFn", fault.ErrCannotDecodeAccount},      // invalid base58 string
	{"anF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXaktvLDj", fault.ErrChecksumMismatch},         // checksum mismatch
	{"7ZpfCEWWU4v3JEAVVHzo7WaiuPeZLMuZ1g6W2dPEGA6g6XEFCz", fault.ErrWrongNetworkForPublicKey}, // wrong network
	{"WjbRFkA9dhmMKnKTuufZ1sVD4E4H1NRnsmwjMKNHHRSCvDm5bXPV", fault.ErrInvalidKeyType},         // undefined key algorithm
	{"YqVxD4vazrrnxnLH2MzCHJedPPz1VKHnKbVfya39nF96ABAYes", fault.ErrNotPublicKey},             // private key
}

// show manually created accounts
// this has to be changed if account.go is modified
// it is used to print the base58Account for testAccount above
func TestValid(t *testing.T) {
	for index, test := range testAccount {
		buffer := []byte{byte(test.algorithm<<4 | 0x01)}
		buffer = append(buffer, test.publicKey...)
		account, err := account.AccountFromBytes(buffer)
		if nil != err {
			t.Errorf("Create account from bytes test: %d failed: %s", index, err)
			continue
		}
		t.Logf("Created account from bytes test: %d result: %s", index, account)
		t.Logf("Created account from bytes test: %d    hex: %x", index, account.Bytes())
	}
}

// From valid base58 string to account
func TestValidBase58(t *testing.T) {
	for index, test := range testAccount {
		account, err := account.AccountFromBase58(test.base58Account)
		if nil != err {
			t.Errorf("Create account from base58 string test: %d failed: %s", index, err)
			continue
		}
		if account.KeyType() != test.algorithm {
			t.Errorf("Create account from base58: %d type: %d  expected: %d", index, account.KeyType(), test.algorithm)
		}
		if !bytes.Equal(account.PublicKeyBytes(), test.publicKey) {
			t.Errorf("Create account from base58: %d pubkey: %x  expected %x", index, account.PublicKeyBytes(), test.publicKey)
		}
		if account.String() != test.base58Account {
			t.Errorf("Create account to base58: %d got: %s  expected %s", index, account, test.base58Account)
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
