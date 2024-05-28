// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/urfave/cli"
)

// identity is required, but not check the config file
func checkName(name string) (string, error) {
	if name == "" {
		return "", fault.IdentityNameIsRequired
	}

	// account names cannot be identities to prevent confusion
	_, err := account.AccountFromBase58(name)
	if err == nil {
		return "", fault.InvalidIdentityName
	}

	return name, nil
}

// check for non-blank file name
func checkFileName(fileName string) (string, error) {
	if fileName == "" {
		return "", fault.FileNameIsRequired
	}

	return fileName, nil
}

// connect is required.
func checkConnect(connect string) (string, error) {
	connect = strings.TrimSpace(connect)
	if connect == "" {
		return "", fault.ConnectIsRequired
	}

	var s []string
	if connect[0] == '[' { // IPv6
		s = strings.Split(connect, "]:")
	} else { // Ipv4 or host
		s = strings.Split(connect, ":")
	}
	if len(s) != 2 {
		return "", fault.ConnectRequiresPortNumberSuffix
	}

	port, err := strconv.Atoi(s[1])
	if err != nil || port < 1 || port > 65535 {
		return "", fault.InvalidPortNumber
	}

	return connect, nil
}

// description is required
func checkDescription(description string) (string, error) {
	if description == "" {
		return "", fault.DescriptionIsRequired
	}

	return description, nil
}

// asset fingerprint is required field
func checkAssetFingerprint(fingerprint string) (string, error) {
	if fingerprint == "" {
		return "", fault.AssetFingerprintIsRequired
	}
	return fingerprint, nil
}

// asset metadata is required field
func checkAssetMetadata(meta string) (string, error) {
	if meta == "" {
		return "", fault.AssetMetadataIsRequired
	}
	meta, err := strconv.Unquote(`"` + meta + `"`)
	if err != nil {
		return "", err
	}
	if len(strings.Split(meta, "\u0000"))%2 == 1 {
		return "", fault.AssetMetadataMustBeMap
	}
	return meta, nil
}

// txid is required field ensure 32 hex bytes
func checkTxId(txId string) (string, error) {
	if len(txId) != 64 {
		return "", fault.TransactionIdIsRequired
	}
	_, err := hex.DecodeString(txId)
	if err != nil {
		return "", err

	}
	return txId, nil
}

// transfer tx is required field
func checkTransferTx(txId string) (string, error) {
	if txId == "" {
		return "", fault.TransactionHexDataIsRequired
	}

	return txId, nil
}

// make sure a seed can be decoded
// strip the "SEED:" prefix if given
func checkSeed(seed string, newSeed bool, testnet bool) (string, error) {

	if newSeed && seed == "" {
		var err error
		seed, err = account.NewBase58EncodedSeedV2(testnet)
		if err != nil {
			return "", err
		}
	}
	seed = strings.TrimPrefix(seed, "SEED:")

	// failed to get a seed
	if seed == "" {
		return "", fault.IncompatibleOptions
	}

	// ensure can decode
	_, err := account.PrivateKeyFromBase58Seed(seed)
	if err != nil {
		return "", err
	}
	return seed, nil
}

// get decrypted identity - prompts for password or uses agent
// only use owner to sign things
func checkOwnerWithPasswordPrompt(name string, config *configuration.Configuration, c *cli.Context) (string, *configuration.Private, error) {
	if name == "" {
		name = config.DefaultIdentity
	}

	var err error

	// get global password items
	agent := c.GlobalString("use-agent")
	clearCache := c.GlobalBool("zero-agent-cache")
	password := c.GlobalString("password")

	// check owner password
	if agent != "" {
		password, err = passwordFromAgent(name, "Password for bitmark-cli", agent, clearCache)
		if err != nil {
			return "", nil, err
		}
	} else if password == "" {
		password, err = promptPassword(name)
		if err != nil {
			return "", nil, err
		}

	}
	owner, err := config.Private(password, name)
	if err != nil {
		return "", nil, err
	}
	return name, owner, nil
}

// recipient is required field convert to an account
// used for any non-signing account process (e.g. provenance listing)
func checkRecipient(c *cli.Context, name string, config *configuration.Configuration) (string, *account.Account, error) {
	recipient := c.String(name)
	if recipient == "" {
		return "", nil, fmt.Errorf("%s is required", name)
	}

	newOwner, err := config.Account(recipient)
	if err != nil {
		return "", nil, err
	}

	return recipient, newOwner, nil
}

// coin address is a required field
func checkCoinAddress(c currency.Currency, address string, testnet bool) (string, error) {
	if address == "" {
		return "", fault.CurrencyAddressIsRequired
	}
	err := c.ValidateAddress(address, testnet)
	return address, err
}

// signature is required field ensure 64 hex bytes
func checkSignature(s string) ([]byte, error) {
	if len(s) != 128 {
		return nil, fault.TransactionIdIsRequired
	}
	h, err := hex.DecodeString(s)
	if err != nil {
		return nil, err

	}
	return h, nil
}

// check if file exists
func checkFileExists(name string) (bool, error) {
	s, err := os.Stat(name)
	if err != nil {
		return false, err
	}
	return s.IsDir(), nil
}
