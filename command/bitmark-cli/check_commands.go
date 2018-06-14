// Copyright (c) 2014-2018 Bitmark Inc.
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
)

var (
	ErrAssetMetadataMustBeMap   = fault.InvalidError("asset metadata must be map")
	ErrRequiredAssetFingerprint = fault.InvalidError("asset fingerprint is required")
	ErrRequiredAssetMetadata    = fault.InvalidError("asset metadata is required")
	ErrRequiredAssetName        = fault.InvalidError("asset name is required")
	ErrRequiredConfigFile       = fault.InvalidError("config file is required")
	ErrRequiredConnect          = fault.InvalidError("connect is required")
	ErrRequiredCurrencyAddress  = fault.InvalidError("currency address is required")
	ErrRequiredDescription      = fault.InvalidError("description is required")
	ErrRequiredFileName         = fault.InvalidError("file name is required")
	ErrRequiredIdentity         = fault.InvalidError("identity is required")
	ErrRequiredPayId            = fault.InvalidError("payment id is required")
	ErrRequiredPublicKey        = fault.InvalidError("public key is required")
	ErrRequiredReceipt          = fault.InvalidError("receipt id is required")
	ErrRequiredTransferTo       = fault.InvalidError("transfer to is required")
	ErrRequiredTransferTx       = fault.InvalidError("transaction hex data is required")
	ErrRequiredTransferTxId     = fault.InvalidError("transaction id is required")
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

func checkNetwork(network string) (string, error) {
	switch network {
	case "":
		network = configuration.DefaultNetwork
	case "bitmark", "live", "production":
		network = "bitmark"
	case "testing", "test":
		network = "testing"
	case "dev", "development", "devel":
		network = "development"
	case "local":
		network = "local"
	default:
		return "", ErrInvalidNetwork
	}
	return network, nil
}

// connect is required.
func checkConnect(connect string) (string, error) {
	if "" == connect {
		return "", ErrRequiredConnect
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

// asset name is required field
func checkAssetName(name string) (string, error) {
	if "" == name {
		return "", ErrRequiredAssetName
	}
	return name, nil
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

func checkAssetQuantity(quantity string) (int, error) {
	if "" == quantity {
		return 1, nil
	}

	i, err := strconv.Atoi(quantity)
	return i, err
}

// transfer txid is required field
func checkTransferTxId(txId string) (string, error) {
	if "" == txId {
		return "", ErrRequiredTransferTxId
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

func checkTransferFrom(from string, config *configuration.Configuration) (*encrypt.IdentityType, error) {
	if "" == from {
		from = config.Default_identity
	}

	return getIdentity(from, config)
}

// transfer to is required field
func checkTransferTo(to string) (string, error) {
	if "" == to {
		return "", ErrRequiredTransferTo
	}
	return to, nil
}

// coin address to is required field
func checkCoinAddress(c currency.Currency, address string, testnet bool) (string, error) {
	if "" == address {
		return "", ErrRequiredCurrencyAddress
	}
	err := c.ValidateAddress(address, testnet)
	return address, err
}

// pay id is required field
func checkPayId(payId string) (string, error) {
	if "" == payId {
		return "", ErrRequiredPayId
	}

	return payId, nil
}

// receipt is required field
func checkReceipt(receipt string) (string, error) {
	if "" == receipt {
		return "", ErrRequiredReceipt
	}

	return receipt, nil
}

func checkRecordCount(count string) (int, error) {
	if "" == count {
		return 20, nil
	}

	i, err := strconv.Atoi(count)
	return i, err
}

// note: this returns apointer to tha actial config.Identity[i]
//       so permanent modifications can be made to the identity
func getIdentity(name string, config *configuration.Configuration) (*encrypt.IdentityType, error) {
	for i, identity := range config.Identity {
		if name == identity.Name {
			return &config.Identity[i], nil
		}
	}

	return nil, ErrNotFoundIdentity
}

// check if file exists
func ensureFileExists(name string) bool {
	_, err := os.Stat(name)
	return nil == err
}
