// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package account_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/fault"
)

// Test privateKey functionality

type privateKeyTest struct {
	algorithm        int
	privateKey       []byte
	base58PrivateKey string
}

type seedTestItem struct {
	seed   string
	addr   string
	priv   string
	isTest bool
}

// Valid privateKey
var testPrivateKey = []privateKeyTest{
	{account.ED25519, decodeHex("95b5a80b4cdbe61c0f3f72cc152d4a4f29bcfd39c9a67e2c7bc6e0e14ec7c7ba55b2988817f7eaec37741b82447163caaa5a9db2b6f0ce722626338e5e3fd7f7"), "AaTfRXLmV59eCFGzBkkzYa1QbuXQBZCiAvjNdnHUaXCFJCyMCxMar6c3Qqaa1mzSPCqPK9XgpkDHcTSCTyAnMnKCHSA2Hz"},
	{account.Nothing, decodeHex("34bc"), "1TG8a64QJ"},
}

// Invalid privateKey
var testInvalidPrivateKeyFromBase58 = []invalid{
	{"3gLJjLSociTmf4kgL3ztUK;tgADFvg9yjXt1jFbEx9KgpEEAFn", fault.ErrCannotDecodePrivateKey},                                   // invalid base58 string
	{"ZxbhGmFUuwUd9XPFoRjPg77T1h29urd2e85pryntETtXCFS3FZ", fault.ErrChecksumMismatch},                                         // checksum mismatch
	{"3iNEz7VJ29DyFeiXGu9gSCUg4K6ykynfPYeyST1AWAti72mpvLd", fault.ErrInvalidKeyType},                                          // undefined key algorithm
	{"anF8SWxSRY5vnN3Bbyz9buRYW1hfCAAZxfbv8Fw9SFXaktvLCj", fault.ErrNotPrivateKey},                                            // public key
	{"AaTfRXLmV59eCFGzBkkzYa1QbuXQBZCiAvjNdnHUaXCFJCyMCxMar6c3Qqaa1mzSPCqPK9XgpkDHcTSCTyAnMnKCHSA2H", fault.ErrNotPrivateKey}, // truncated
	{"aTfRXLmV59eCFGzBkkzYa1QbuXQBZCiAvjNdnHUaXCFJCyMCxMar6c3Qqaa1mzSPCqPK9XgpkDHcTSCTyAnMnKCHSA2Hz", fault.ErrNotPrivateKey}, // truncated
}

// valid seed
var validSeedTestItems = []seedTestItem{
	{"5XEECqhR7QBkJezUJiUJBmHaSmffDfVN5atuLnQBHnvfxbsWHuBfQLw", "ajsDToCYSuK9rjSKGU6pwKGHahybu3DJ42DYbXRgHxS3Yc6CFC", "e6d85658b86242d45b52d9421736427ef22edda12c8790408c09ec3c9e356e755b4d99cc95cec16a3d489c94ba33d7fd6705c6cd3a6495c264e188b1985f4249", false},
	{"5XEECtzqJYokJbDkLzPMqNEF1Eo5qfGPqhbb4pGeuj2igeEMYraCcJ1", "fGcv38F4ucFwvwnepNYYDQt3eDjRaoVtLCdofMYGUENboXVQzx", "83fb4107766d5fd66d0648dcafbc6e77b24d8cced42940ae3a62bb98810e189bafeabdcd58645fa58c70fed58fea0ca95682ca4e20a4aae44319383865383b21", true},
	{"9J877LVjhr3Xxd2nGzRVRVNUZpSKJF4TH", "f7nuKToBByL3jEcArZWoB9PJ8MVmGPjrYkW88v3Yw8p7G5Sxhy", "4534075cbcfc6ada1bb6b9e53d53f72341746031d9d17a3089a117766e7cda9e9bdf52f23deb941ea23cec982c24a5c811d321e71f6df56508bd511f66311e06", true},
	{"9J876mP7wDJ6g5P41eNMN8N3jo9fycDs2", "fXXHGtCdFPuQvNhJ4nDPKCdwPxH7aSZ4842n2katZi319NsaCs", "b0e77f0a27390a00e82c07d6d228999019dce17aa3fbc1958629a7a47bc1cf6dd1c177ef358e9d1f0d4b09328cc1213e8d3580703aee51ccf97e482be977f7bc", true},
}

// invalid base58 seed
var invalidBase58Seeds = []invalid{
	{"5XEECqhR7QBkJezUJiUJBmHaSmffDfVN5atuLnQBHnvfxbsWHuBfQ", fault.ErrInvalidSeedLength},
	{"9J877LVjhr3Xxd2nGzRVRVNUZpSKJF4THGaf", fault.ErrInvalidSeedLength},
	{"5XEECqhR7QBkJezUJiUJBmHaSmffDfVN5atuLnQBHnvfxbsWHuBfQkw", fault.ErrChecksumMismatch},
	{"9J877LVjhr3Xxd2nGzRVRVNUZpSKJF4TG", fault.ErrChecksumMismatch},
	{"9J3KBhE3TBmVfpH4Xcw7hXsAxDCgdgvdg", fault.ErrInvalidSeedHeader},
	{"5XBcj8Cz1Aj5yciJkivUrfYUbBk1LfgtfQ9oX8wsrA4QmmYw1miJSCE", fault.ErrInvalidSeedHeader},
}

// show manually created private keys
// this has to be changed if private.go is modified
// it is used to print the base58PrivateKey for testPrivateKey above
func TestPrivateValid(t *testing.T) {
loop:
	for index, test := range testPrivateKey {
		buffer := []byte{byte(test.algorithm << 4)}
		buffer = append(buffer, test.privateKey...)
		privateKey, err := account.PrivateKeyFromBytes(buffer)
		if nil != err {
			t.Errorf("%d: Create privateKey from bytes failed: %s", index, err)
			continue loop
		}
		t.Logf("%d: result: %s", index, privateKey)
		t.Logf("%d:    hex: %x", index, privateKey.Bytes())
	}
}

// From valid base58 string to privateKey
func TestPrivateValidBase58(t *testing.T) {
loop:
	for index, test := range testPrivateKey {
		prv, err := account.PrivateKeyFromBase58(test.base58PrivateKey)
		if nil != err {
			t.Errorf("%d: from base58 error: %s", index, err)
			continue loop
		}
		if prv.KeyType() != test.algorithm {
			t.Errorf("%d: from base58 type: %d  expected: %d", index, prv.KeyType(), test.algorithm)
		}
		if !bytes.Equal(prv.PrivateKeyBytes(), test.privateKey) {
			t.Errorf("%d: from base58 pubkey: %x  expected %x", index, prv.PrivateKeyBytes(), test.privateKey)
		}
		if prv.String() != test.base58PrivateKey {
			t.Errorf("%d: to base58: got: %s  expected %s", index, prv, test.base58PrivateKey)
		}

		// test unmarshal JSON
		j := `"` + test.base58PrivateKey + `"`
		var a account.PrivateKey
		err = json.Unmarshal([]byte(j), &a)
		if nil != err {
			t.Errorf("%d: from JSON string error: %s", index, err)
			continue loop
		}
		t.Logf("%d: from JSON: %#v", index, a)

		buffer, _ := json.Marshal(a)
		t.Logf("%d: privateKey to JSON: %s", index, buffer)
		if j != string(buffer) {
			t.Errorf("%d: marshal JSON:failed: expected %s  actual: %s", index, j, buffer)
		}

	}
}

// Test invalid privateKey parsing
// From privateKey base58 encoded to privateKey
func TestPrivateInvalidBase58(t *testing.T) {
	for index, test := range testInvalidPrivateKeyFromBase58 {
		_, err := account.PrivateKeyFromBase58(test.str)
		if test.err != err {
			t.Errorf("invalid base58 string: %d failed: expected: %q actual: %q", index, test.err, err)
		}
	}
}

// Test valid base58 seed parsing
func TestPrivateKeyFromBase58Seed(t *testing.T) {
	for _, item := range validSeedTestItems {
		checkSeed(t, item.seed, item.addr, item.priv, item.isTest)
	}
}

// Test invalid base58 seed parsing
func TestPrivateKeyFromInvalidBase58Seed(t *testing.T) {
	for _, testItem := range invalidBase58Seeds {
		_, err := account.PrivateKeyFromBase58Seed(testItem.str)
		if expectedErr := testItem.err; err != expectedErr {
			t.Errorf("test private key from base58 seed: expected: %s, actual: %s", expectedErr, err)
		}
	}
}

func checkSeed(t *testing.T, seed string, address string, privateKeyHex string, isTest bool) {

	privateKey := decodeHex(privateKeyHex)

	k, err := account.PrivateKeyFromBase58Seed(seed)
	if nil != err {
		t.Fatalf("seed error: %s", err)
	}

	if isTest != k.IsTesting() {
		t.Errorf("network is testing: actual: %t, expected: %t", k.IsTesting(), isTest)
	}

	actual := k.PrivateKeyBytes()
	if !bytes.Equal(privateKey, actual) {
		t.Fatalf("invalid private key: expected: %x  actual: %x", privateKey, actual)
	}

	accExpected, err := account.AccountFromBase58(address)
	if nil != err {
		t.Fatalf("account from base58 error: %s", err)
	}

	accActual := k.Account()
	if nil == accActual {
		t.Fatal("account from private key returned nil")
	}

	if !bytes.Equal(accActual.PublicKeyBytes(), accExpected.PublicKeyBytes()) {
		t.Errorf("public key expected: %x", accExpected.PublicKeyBytes())
		t.Errorf("public key actual:   %x", accActual.PublicKeyBytes())
	}

	if !bytes.Equal(accActual.Bytes(), accExpected.Bytes()) {
		t.Errorf("bytes expected: %x", accExpected.Bytes())
		t.Errorf("bytes actual:   %x", accActual.Bytes())
	}

	if accExpected.String() != accActual.String() {
		t.Errorf("invalid account: expected: %q", accExpected)
		t.Errorf("invalid account: actual:   %q", accActual)
	}
}
