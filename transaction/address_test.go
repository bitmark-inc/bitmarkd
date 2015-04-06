// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package transaction_test

import (
	"bytes"
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"testing"
)

// Test address functionality

type addressTest struct {
	algorithm     int
	publicKey     []byte
	base58Address string
}

// Valid address
var testAddress = []addressTest{
	{transaction.ED25519, decodeHex("60b3c6e20cfff7091a86488b1656b96ec0a2f69907e2c035175918f42c37d72e"), "anF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXanoTNEq"},
	{transaction.Nothing, decodeHex("12fa"), "3MvzKPwWD"},
}

// Invalid address
var testInvalidAddressFromBase58 = []string{
	"3gLJjLSociTmf4kgL3ztUK;tgADFvg9yjXt1jFbEx9KgpEEAFn",   // invalid base58 string
	"7ZpfCEWWU4v3JEAVVHzo7WaiuPeZLMuZ1g6W2dPEGA6g6XEFCz",   // checksum mismatch
	"WjbRFkA9dhmMKnKTuufZ1sVD4E4H1NRnsmwjMKNHHRSCvDm5bXPV", // undefined key algorithm
	"1jb8VtQxC3EdqV3mkRzw9iFyZYVcDqHC6TmmaZhFJ8wCA2yDQC",   // Send private key instead
}

func TestAddress(t *testing.T) {

	// show manually created addresses
	// this has to be changed if address.go is modified
	// it is used to print the base58Address for testAddress above
	for index, test := range testAddress {
		buffer := []byte{byte(test.algorithm<<4 | 0x01)}
		buffer = append(buffer, test.publicKey...)
		address, err := transaction.AddressFromBytes(buffer)
		if nil != err {
			t.Errorf("Create address from bytes test: %d failed: %s", index, err)
			continue
		}
		t.Logf("Created address from bytes test: %d result: %s", index, address)
	}

	// From address base58 encoded to address
	for index, test := range testAddress {
		address, err := transaction.AddressFromBase58(test.base58Address)
		if nil != err {
			t.Errorf("Create address from base58 string test: %d failed: %s", index, err)
			continue
		}
		if address.KeyType() != test.algorithm {
			t.Errorf("Create address from base58: %d type: %d  expected: %d", index, address.KeyType(), test.algorithm)
		}
		if !bytes.Equal(address.PublicKeyBytes(), test.publicKey) {
			t.Errorf("Create address from base58: %d pubkey: %x  expected %x", index, address.PublicKeyBytes(), test.publicKey)
		}
		if address.String() != test.base58Address {
			t.Errorf("Create address to base58: %d got: %s  expected %s", index, address, test.base58Address)
		}
	}

	// Test invalid address parsing
	// From address base58 encoded to address
	for index, test := range testInvalidAddressFromBase58 {
		_, err := transaction.AddressFromBase58(test)
		if nil == err {
			t.Errorf("Create address from invalid base58 string: %d failed: did not error on invalid input", index)
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
