// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"os"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

const (
	ErrAssetMetadataMustBeMap   = fault.InvalidError("asset metadata must be map")
	ErrRequiredAssetFingerprint = fault.InvalidError("asset fingerprint is required")
	ErrRequiredAssetMetadata    = fault.InvalidError("asset metadata is required")
	ErrRequiredAssetName        = fault.InvalidError("asset name is required")
	ErrRequiredConnect          = fault.InvalidError("connect is required use: HOST:PORT or IPv4:PORT or [IPv6]:PORT")
	ErrRequiredConnectPort      = fault.InvalidError("connect requires :PORT-NUMBER suffix")
	ErrRequiredCurrencyAddress  = fault.InvalidError("currency address is required")
	ErrRequiredDescription      = fault.InvalidError("description is required")
	ErrRequiredFileName         = fault.InvalidError("file name is required")
	ErrRequiredIdentity         = fault.InvalidError("identity is required")
	ErrRequiredPayId            = fault.InvalidError("payment id is required")
	ErrRequiredPublicKey        = fault.InvalidError("public key is required")
	ErrRequiredReceipt          = fault.InvalidError("receipt id is required")
	ErrRequiredTransferTo       = fault.InvalidError("transfer to is required")
	ErrRequiredTransferTx       = fault.InvalidError("transaction hex data is required")
	ErrRequiredTxId             = fault.InvalidError("transaction id is required")
)

// identity is required, but not check the config file
func checkName(name string) (string, error) {
	if "" == name {
		return "", ErrRequiredIdentity
	}

	return name, nil
}

// check for non-blank file name
func checkFileName(fileName string) (string, error) {
	if "" == fileName {
		return "", ErrRequiredFileName
	}

	return fileName, nil
}

// connect is required.
func checkConnect(connect string) (string, error) {
	connect = strings.TrimSpace(connect)
	if "" == connect {
		return "", ErrRequiredConnect
	}

	s := []string{}

	if '[' == connect[0] { // IPv6
		s = strings.Split(connect, "]:")
	} else { // Ipv4 or host
		s = strings.Split(connect, ":")
	}
	if 2 != len(s) {
		return "", ErrRequiredConnectPort
	}

	port, err := strconv.Atoi(s[1])
	if nil != err || port < 1 || port > 65535 {
		return "", ErrRequiredConnectPort
	}

	return connect, nil
}

// description is required
func checkDescription(description string) (string, error) {
	if "" == description {
		return "", ErrRequiredDescription
	}

	return description, nil
}

// private key is optional,
// if present must be either 64 or 128 hex chars
// or SEED:<base58-seed>
func checkOptionalKey(key string) (string, error) {
	if "" == key {
		return "", nil
	}
	if strings.HasPrefix(key, "SEED:") {
		return key, nil
	}
	k, err := hex.DecodeString(key)
	if nil != err {
		return "", err
	}
	switch len(k) {
	case encrypt.PrivateKeySize: // have the full key (private + public)
	case encrypt.PublicKeyOffset: // just have the private part
	default:
		return "", ErrKeyLength
	}
	return key, nil
}

// prublic key is require,
// if present must 64 hex chars
func checkPublicKey(key string) (string, error) {
	if "" == key {
		return "", ErrRequiredPublicKey

	}
	k, err := hex.DecodeString(key)
	if nil != err {
		return "", err
	}
	switch len(k) {
	case encrypt.PublicKeySize: // have the full key
	default:
		return "", ErrKeyLength
	}
	return key, nil
}

func checkIdentity(name string, config *configuration.Configuration) (*encrypt.IdentityType, error) {
	if "" == name {
		return nil, ErrRequiredIdentity
	}

	return getIdentity(name, config)
}

// asset fingerprint is required field
func checkAssetFingerprint(fingerprint string) (string, error) {
	if "" == fingerprint {
		return "", ErrRequiredAssetFingerprint
	}
	return fingerprint, nil
}

// asset metadata is required field
func checkAssetMetadata(meta string) (string, error) {
	if "" == meta {
		return "", ErrRequiredAssetMetadata
	}
	meta, err := strconv.Unquote(`"` + meta + `"`)
	if nil != err {
		return "", err
	}
	if 1 == len(strings.Split(meta, "\u0000"))%2 {
		return "", ErrAssetMetadataMustBeMap
	}
	return meta, nil
}

// txid is required field ensure 32 hex bytes
func checkTxId(txId string) (string, error) {
	if 64 != len(txId) {
		return "", ErrRequiredTxId
	}
	_, err := hex.DecodeString(txId)
	if nil != err {
		return "", err

	}
	return txId, nil
}

// transfer tx is required field
func checkTransferTx(txId string) (string, error) {
	if "" == txId {
		return "", ErrRequiredTransferTx
	}

	return txId, nil
}

// contains private and public keys
func checkTransferFrom(from string, config *configuration.Configuration) (*encrypt.IdentityType, error) {
	if "" == from {
		from = config.DefaultIdentity
	}

	return getIdentity(from, config)
}

// transfer to is required field but only has a public key
func checkTransferTo(to string, config *configuration.Configuration) (string, *keypair.KeyPair, error) {
	if "" == to {
		return "", nil, ErrRequiredTransferTo
	}

	newOwnerKeyPair, err := encrypt.PublicKeyFromString(to, config.Identities, config.TestNet)
	if nil != err {
		return "", nil, err
	}

	return to, newOwnerKeyPair, nil
}

// coin address to is required field
func checkCoinAddress(c currency.Currency, address string, testnet bool) (string, error) {
	if "" == address {
		return "", ErrRequiredCurrencyAddress
	}
	err := c.ValidateAddress(address, testnet)
	return address, err
}

// signature is required field ensure 64 hex bytes
func checkSignature(s string) ([]byte, error) {
	if 128 != len(s) {
		return nil, ErrRequiredTxId
	}
	h, err := hex.DecodeString(s)
	if nil != err {
		return nil, err

	}
	return h, nil
}

// note: this returns a pointer to the actual config.Identity[i]
//       so permanent modifications can be made to the identity
func getIdentity(name string, config *configuration.Configuration) (*encrypt.IdentityType, error) {
	for i, identity := range config.Identities {
		if name == identity.Name {
			return &config.Identities[i], nil
		}
	}

	return nil, ErrNotFoundIdentity
}

// check if file exists
func checkFileExists(name string) (bool, error) {
	s, err := os.Stat(name)
	if nil != err {
		return false, err
	}
	return s.IsDir(), nil
}
