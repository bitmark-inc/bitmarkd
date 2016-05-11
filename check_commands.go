// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"github.com/bitmark-inc/exitwithstatus"
	"os"
	"strconv"
)

// config is required
func checkConfigFile(file string) (string, error) {
	if "" == file {
		return "", fault.ErrRequiredConfig
	}

	file = os.ExpandEnv(file)
	return file, nil
}

// identity is required, but not check the config file
func checkName(name string) (string, error) {
	if "" == name {
		return "", fault.ErrRequiredIdentity
	}

	return name, nil
}

func checkNetwork(network string) string {
	if "" == network {
		network = configuration.DefaultNetwork
	} else {
		if "testing" != network && "bitmark" != network {
			exitwithstatus.Message("Error: Wrong Network value [bitmark | testing]: %s", network)
		}
	}
	return network
}

// connect is required.
func checkConnect(connect string) (string, error) {
	if "" == connect {
		return "", fault.ErrRequiredConnect
	}

	return connect, nil
}

// description is required
func checkDescription(description string) (string, error) {
	if "" == description {
		return "", fault.ErrRequiredDescription
	}

	return description, nil
}

func checkIdentity(name string, config *configuration.Configuration) (*configuration.IdentityType, error) {
	if "" == name {
		return nil, fault.ErrRequiredIdentity
	}

	return getIdentity(name, config.Identities)
}

// asset name is required field
func checkAssetName(name string) (string, error) {
	if "" == name {
		return "", fault.ErrRequiredAssetName
	}
	return name, nil
}

// asset description is required field
func checkAssetDescription(desc string) (string, error) {
	if "" == desc {
		return "", fault.ErrRequiredAssetDescription
	}
	return desc, nil
}

// asset fingerprint is required field
func checkAssetFingerprint(fingerprint string) (string, error) {
	if "" == fingerprint {
		return "", fault.ErrRequiredAssetFingerprint
	}
	return fingerprint, nil
}

func checkAssetQuantity(quantity string) (int, error) {
	if "" == quantity {
		return 1, nil
	}

	i, err := strconv.Atoi(quantity)
	return i, err
}

// transfer tx_id is required field
func checkTransferTxId(txId string) (string, error) {
	if "" == txId {
		return "", fault.ErrRequiredTransferTxId
	}

	return txId, nil
}

// transfer to is required field
func checkTransferTo(to string, identities []configuration.IdentityType) (*configuration.IdentityType, error) {
	if "" == to {
		return nil, fault.ErrRequiredTransferTo
	}
	return getIdentity(to, identities)
}

func checkTransferFrom(from string, config *configuration.Configuration) (*configuration.IdentityType, error) {
	if "" == from {
		from = config.Default_identity
	}

	return getIdentity(from, config.Identities)
}

func checkAndGetConfig(path string) (*configuration.Configuration, error) {
	configFile, err := checkConfigFile(path)
	if nil != err {
		return nil, err
	}

	configuration, err := configuration.GetConfiguration(configFile)
	if nil != err {
		return nil, err
	}

	return configuration, nil

}

func getIdentity(name string, identities []configuration.IdentityType) (*configuration.IdentityType, error) {
	for _, identity := range identities {
		if name == identity.Name {
			return &identity, nil
		}
	}

	return nil, fault.ErrNotFoundIdentity
}

// check if file exists
func ensureFileExists(name string) bool {
	_, err := os.Stat(name)
	return nil == err
}
