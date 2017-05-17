// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/codegangsta/cli"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"os"
	"strings"
)

// version byte prefix for fignerprint file
const (
	fingerprintVersion byte = 0x01
)

func runGenerate(c *cli.Context, globals globalFlags) {
	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	// flag to indicate testnet keys
	testnet := "bitmark" != configData.Network

	rawKeyPair, _, err := makeRawKeyPair(testnet)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	if b, err := json.MarshalIndent(rawKeyPair, "", "  "); nil != err {
		exitwithstatus.Message("Error: %s", err)
	} else {
		fmt.Printf("%s\n", b)
	}
}

func runSetup(c *cli.Context, globals globalFlags) {
	configFile, err := checkConfigFile(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	// do not run setup if there is an existing configuration
	if ensureFileExists(configFile) {
		exitwithstatus.Message("Error: not overwriting existing configuration: %q", configFile)
	}

	name, err := checkName(globals.identity)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	network := checkNetwork(c.String("network"))

	connect, err := checkConnect(c.String("connect"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	description, err := checkDescription(c.String("description"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	// optional existing hex key value
	privateKey, err := checkOptionalKey(c.String("privateKey"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("config: %s\n", configFile)
		fmt.Printf("network: %s\n", network)
		fmt.Printf("connect: %s\n", connect)
		fmt.Printf("identity: %s\n", name)
		fmt.Printf("description: %s\n", description)
		fmt.Println()
	}

	// Create the folder hierarchy for configuration if not existing
	folderIndex := strings.LastIndex(configFile, "/")
	if folderIndex >= 0 {
		configDir := configFile[:folderIndex]
		if !ensureFileExists(configDir) {
			if err := os.MkdirAll(configDir, 0755); nil != err {
				exitwithstatus.Message("Error: %s", err)
			}
		}
	}
	configData := &configuration.Configuration{
		Default_identity: name,
		Network:          network,
		Connect:          connect,
		Identity:         make([]configuration.IdentityType, 0),
	}

	// flag to indicate testnet keys
	testnet := "bitmark" != configData.Network

	if !addIdentity(configData, name, description, privateKey, globals.password, testnet) {
		exitwithstatus.Message("Error: Setup failed")
	}
	err = configuration.Save(configFile, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}
}

func runAdd(c *cli.Context, globals globalFlags) {

	configFile, err := checkConfigFile(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	name, err := checkName(globals.identity)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	description, err := checkDescription(c.String("description"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	// optional existing hex key value
	privateKey, err := checkOptionalKey(c.String("privateKey"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("config: %s\n", configFile)
		fmt.Printf("identity: %s\n", name)
		fmt.Printf("description: %s\n", description)
		fmt.Println()
	}
	// flag to indicate testnet keys
	testnet := "bitmark" != configData.Network

	if !addIdentity(configData, name, description, privateKey, globals.password, testnet) {
		exitwithstatus.Message("Error: add failed")
	}
	err = configuration.Save(configFile, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}
}

func runCreate(c *cli.Context, globals globalFlags, batchMode bool) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	issuer, err := checkIdentity(globals.identity, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	assetName, err := checkAssetName(c.String("asset"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	fingerprint, err := checkAssetFingerprint(c.String("fingerprint"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	metadata, err := checkAssetMetadata(c.String("metadata"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	quantity, err := checkAssetQuantity(c.String("quantity"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("issuer: %s\n", issuer.Name)
		fmt.Printf("assetName: %q\n", assetName)
		fmt.Printf("fingerprint: %q\n", fingerprint)
		fmt.Printf("metadata:\n")
		m := strings.Split(metadata, "\u0000")
		for i := 0; i < len(m); i += 2 {
			fmt.Printf("  %q: %q\n", m[i], m[i+1])
		}
		fmt.Printf("quantity: %d\n", quantity)
	}

	var registrant *KeyPair

	// check password
	if "" != globals.agent {
		password, err := passwordFromAgent(issuer.Name, "Create Bitmark", globals.agent, globals.clearCache)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
		registrant, err = verifyPassword(password, issuer)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else if "" != globals.password {
		registrant, err = verifyPassword(globals.password, issuer)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else {
		registrant, err = promptAndCheckPassword(issuer)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	}
	// just in case some internal breakage
	if nil == registrant {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configData.Connect,
		network:  configData.Network,
	}

	assetConfig := assetData{
		name:        assetName,
		metadata:    metadata,
		quantity:    quantity,
		fingerprint: fingerprint,
		registrant:  registrant,
	}

	if batchMode {
		outputFilename := c.String("output")

		err = batch(bitmarkRpcConfig, assetConfig, outputFilename, verbose)
		if nil != err {
			exitwithstatus.Message("Issue error: %s", err)
		}
	} else {
		err = issue(bitmarkRpcConfig, assetConfig, verbose)
		if nil != err {
			exitwithstatus.Message("Issue error: %s", err)
		}
	}
}

func runTransfer(c *cli.Context, globals globalFlags) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	txId, err := checkTransferTxId(c.String("txid"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	to, err := checkTransferTo(c.String("receiver"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	from, err := checkTransferFrom(globals.identity, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("txid: %s\n", txId)
		fmt.Printf("receiver: %s\n", to)
		fmt.Printf("sender: %s\n", from.Name)
	}

	var ownerKeyPair *KeyPair
	// check owner password
	if "" != globals.agent {
		password, err := passwordFromAgent(from.Name, "Transfer Bitmark", globals.agent, globals.clearCache)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
		ownerKeyPair, err = verifyPassword(password, from)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else if "" != globals.password {
		ownerKeyPair, err = verifyPassword(globals.password, from)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else {
		ownerKeyPair, err = promptAndCheckPassword(from)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}

	}
	// just in case some internal breakage
	if nil == ownerKeyPair {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	var newOwnerKeyPair *KeyPair

	// ***** FIX THIS: possibly add base58 keys @@@@@
	newPublicKey, err := hex.DecodeString(to)
	if nil != err {

		newOwnerKeyPair, err = publicKeyFromIdentity(to, configData.Identity)
		if nil != err {
			exitwithstatus.Message("receiver identity error: %s", err)
		}
	} else {
		newOwnerKeyPair = &KeyPair{}
		if len(newPublicKey) != publicKeySize {
			exitwithstatus.Message("hex public key must be %d bytes", publicKeySize)
		}
		newOwnerKeyPair.PublicKey = newPublicKey
	}
	// just in case some internal breakage
	if nil == newOwnerKeyPair {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configData.Connect,
		network:  configData.Network,
	}

	link, err := txIdFromString(txId)
	if nil != err {
		exitwithstatus.Message("Transfer TxId error: %s", err)
	}

	transferConfig := transferData{
		owner:    ownerKeyPair,
		newOwner: newOwnerKeyPair,
		txId:     link,
	}

	err = transfer(bitmarkRpcConfig, transferConfig, verbose)
	if nil != err {
		exitwithstatus.Message("Transfer error: %s", err)
	}
}

func runProvenance(c *cli.Context, globals globalFlags) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	txId, err := checkTransferTxId(c.String("txid"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	count, err := checkRecordCount(c.String("count"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("txid: %s\n", txId)
		fmt.Printf("count: %d\n", count)
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configData.Connect,
		network:  configData.Network,
	}

	provenanceConfig := provenanceData{
		txId:  txId,
		count: count,
	}

	err = provenance(bitmarkRpcConfig, provenanceConfig, verbose)
	if nil != err {
		exitwithstatus.Message("Provenance error: %s", err)
	}
}

func runTransactionStatus(c *cli.Context, globals globalFlags) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	txId, err := checkTransferTxId(c.String("txid"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("txid: %s\n", txId)
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configData.Connect,
		network:  configData.Network,
	}

	statusConfig := transactionStatusData{
		txId: txId,
	}

	err = transactionStatus(bitmarkRpcConfig, statusConfig, verbose)
	if nil != err {
		exitwithstatus.Message("Transaction Status error: %s", err)
	}
}

func runPublicKeyDisplay(c *cli.Context, globals globalFlags) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	// flag to indicate testnet keys
	testnet := "bitmark" != configData.Network

	publicKey, err := checkPublicKey(c.String("publickey"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("publicKey: %s\n", publicKey)
	}

	account, err := accountFromHexPublicKey(publicKey, testnet)
	if nil != err {
		exitwithstatus.Message("Transaction Status error: %s", err)
	}

	result := struct {
		Hex    string `json:"hex"`
		Base58 string `json:"account"`
	}{
		Hex:    publicKey,
		Base58: account.String(),
	}

	printJson("", result)
}

func runInfo(c *cli.Context, globals globalFlags) {

	infoConfig, err := configuration.GetInfoConfiguration(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	// add base58 Bitmark Account to output structure
	for i, id := range infoConfig.Identity {
		pub, err := hex.DecodeString(id.Public_key)
		if nil != err {
			exitwithstatus.Message("Error: Get configuration failed: %s", err)
		}

		keyPair := &KeyPair{
			PublicKey: pub,
		}
		infoConfig.Identity[i].Account = makeAddress(keyPair, infoConfig.Network).String()
	}

	printJson("", infoConfig)
}

func runBitmarkInfo(c *cli.Context, globals globalFlags) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	verbose := globals.verbose

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configData.Connect,
		network:  configData.Network,
	}

	if err := bitmarkInfo(bitmarkRpcConfig, verbose); nil != err {
		exitwithstatus.Message("Error: Get info failed: %s", err)
	}
}

func runFingerprint(c *cli.Context, globals globalFlags) {

	fileName, err := checkFileName(c.String("file"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("file: %s\n", fileName)
	}

	file, err := os.Open(fileName)
	if nil != err {
		exitwithstatus.Message("cannot open: %q  error: %s", fileName, err)
	}

	data, err := ioutil.ReadAll(file)

	fingerprint := sha3.Sum512(data)

	fmt.Printf("fingerprint: %02x%x\n", fingerprintVersion, fingerprint)
}

func runSign(c *cli.Context, globals globalFlags) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	fileName, err := checkFileName(c.String("file"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	from, err := checkTransferFrom(globals.identity, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("file: %s\n", fileName)
		fmt.Printf("signer: %s\n", from.Name)
	}

	var ownerKeyPair *KeyPair
	// check owner password
	if "" != globals.agent {
		password, err := passwordFromAgent(from.Name, "Transfer Bitmark", globals.agent, globals.clearCache)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
		ownerKeyPair, err = verifyPassword(password, from)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else if "" != globals.password {
		ownerKeyPair, err = verifyPassword(globals.password, from)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}
	} else {
		ownerKeyPair, err = promptAndCheckPassword(from)
		if nil != err {
			exitwithstatus.Message("Error: %s", err)
		}

	}
	// just in case some internal breakage
	if nil == ownerKeyPair {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	file, err := os.Open(fileName)
	if nil != err {
		exitwithstatus.Message("cannot open: %q  error: %s", fileName, err)
	}

	data, err := ioutil.ReadAll(file)

	signature := ed25519.Sign(ownerKeyPair.PrivateKey, data)
	s := hex.EncodeToString(signature)
	fmt.Printf("signature: %q\n", s)
}
