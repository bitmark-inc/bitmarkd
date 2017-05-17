// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/codegangsta/cli"
	"net/rpc/jsonrpc"
)

func addIdentity(configs *configuration.Configuration, name string, description string, privateKeyStr string, password string, testnet bool) bool {

	for _, identity := range configs.Identity {
		if name == identity.Name {
			fmt.Printf("identity: %q already exists\n", name)
			return false
		}
	}

	if "" == password {
		var err error
		// prompt password and pwd confirm for private key encryption
		password, err = promptPasswordReader()
		if nil != err {
			fmt.Printf("input password fail: %s\n", err)
			return false
		}
	}

	encrypted, privateKeyConfig, err := makeKeyPair(privateKeyStr, password, testnet)
	if nil != err {
		fmt.Printf("error generating key pair: %s\n", err)
		return false
	}

	identity := configuration.IdentityType{
		Name:               name,
		Description:        description,
		Public_key:         encrypted.PublicKey,
		Seed:               encrypted.EncryptedSeed,
		Private_key:        encrypted.EncryptedPrivateKey,
		Private_key_config: *privateKeyConfig,
	}
	configs.Identity = append(configs.Identity, identity)

	return true
}

func issue(rpcConfig bitmarkRPC, assetConfig assetData, verbose bool) error {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	// make asset
	assetIndex, err := makeAsset(client, rpcConfig.network, assetConfig, verbose)
	if nil != err {
		return err
	}

	// make Issues
	issueConfig := issueData{
		issuer:     assetConfig.registrant,
		assetIndex: assetIndex,
		quantity:   assetConfig.quantity,
	}
	response, err := doIssues(client, rpcConfig.network, issueConfig, verbose)
	if nil != err {
		return err
	}
	if verbose {
		fmt.Printf("Result:\n")
	}
	printJson("", response)

	return nil
}

func transfer(rpcConfig bitmarkRPC, transferConfig transferData, verbose bool) error {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	response, err := doTransfer(client, rpcConfig.network, transferConfig, verbose)
	if nil != err {
		return err
	}
	if verbose {
		fmt.Printf("Result:\n")
	}
	printJson("", response)

	return nil
}

func provenance(rpcConfig bitmarkRPC, provenanceConfig provenanceData, verbose bool) error {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	response, err := doProvenance(client, rpcConfig.network, provenanceConfig, verbose)
	if nil != err {
		return err
	}
	if verbose {
		fmt.Printf("Result:\n")
	}
	printJson("", response)

	return nil
}

func transactionStatus(rpcConfig bitmarkRPC, statusConfig transactionStatusData, verbose bool) error {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	response, err := doTransactionStatus(client, rpcConfig.network, statusConfig, verbose)
	if nil != err {
		return err
	}
	if verbose {
		fmt.Printf("Result:\n")
	}
	printJson("", response)

	return nil
}

func bitmarkInfo(rpcConfig bitmarkRPC, verbose bool) error {
	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	response, err := getBitmarkInfo(client, verbose)
	if nil != err {
		return err
	}
	if verbose {
		fmt.Printf("Result:\n")
	}
	printJson("", response)

	return nil
}

func getDefaultRawKeyPair(c *cli.Context, globals globalFlags) {
	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	identity, err := checkTransferFrom(globals.identity, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	var keyPair *KeyPair

	// check owner password
	if "" == globals.password {
		keyPair, err = promptAndCheckPassword(identity)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else {
		keyPair, err = verifyPassword(globals.password, identity)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	}
	//just in case some internal breakage
	if nil == keyPair {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	type KeyPairDisplay struct {
		Account    *account.Account    `json:"account"`
		PrivateKey *account.PrivateKey `json:"private_key"`
		KeyPair    RawKeyPair          `json:"raw"`
	}
	output := KeyPairDisplay{
		Account:    makeAddress(keyPair, configData.Network),
		PrivateKey: makePrivateKey(keyPair, configData.Network),
		KeyPair: RawKeyPair{
			Seed:       keyPair.Seed,
			PublicKey:  hex.EncodeToString(keyPair.PublicKey[:]),
			PrivateKey: hex.EncodeToString(keyPair.PrivateKey[:]),
		},
	}
	if b, err := json.MarshalIndent(output, "", "  "); nil != err {
		exitwithstatus.Message("Error: %s", err)
	} else {
		fmt.Printf("%s\n", b)
	}
}

func changePassword(c *cli.Context, globals globalFlags) {
	configFile, err := checkConfigFile(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	// flag to indicate testnet keys
	testnet := "bitmark" != configData.Network

	identity, err := checkTransferFrom(globals.identity, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	var keyPair *KeyPair

	// check owner password
	if "" == globals.password {
		keyPair, err = promptAndCheckPassword(identity)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else {
		keyPair, err = verifyPassword(globals.password, identity)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	}
	//just in case some internal breakage
	if nil == keyPair {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	// prompt new password and pwd confirm for private key encryption
	newPassword, err := promptPasswordReader()
	if nil != err {
		exitwithstatus.Message("input password fail: %s", err)
	}

	input := ""
	if 0 == len(keyPair.Seed) {
		input = hex.EncodeToString(keyPair.PrivateKey[:])
	} else {
		input = "SEED:" + keyPair.Seed
	}

	encrypted, privateKeyConfig, err := makeKeyPair(input, newPassword, testnet)
	if nil != err {
		exitwithstatus.Message("make key pair error: %s", err)
	}
	if encrypted.PublicKey != identity.Public_key {
		exitwithstatus.Message("public key was modified", err)
	}
	identity.Seed = encrypted.EncryptedSeed
	identity.Private_key = encrypted.EncryptedPrivateKey
	identity.Private_key_config = *privateKeyConfig

	err = configuration.Save(configFile, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}
}
