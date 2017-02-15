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
	"os"
	"strings"
)

type globalFlags struct {
	verbose    bool
	config     string
	identity   string
	password   string
	agent      string
	clearCache bool
}

func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	globals := globalFlags{}

	app := cli.NewApp()
	app.Name = "bitmark-cli"
	// app.Usage = ""
	app.Version = Version
	app.HideVersion = true
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "verbose, v",
			Usage:       " verbose result",
			Destination: &globals.verbose,
		},
		cli.StringFlag{
			Name:        "config, c",
			Value:       "",
			Usage:       "bitmark-cli configuration directory",
			Destination: &globals.config,
		},
		cli.StringFlag{
			Name:        "identity, i",
			Value:       "",
			Usage:       " identity name [default identity]",
			Destination: &globals.identity,
		},
		cli.StringFlag{
			Name:        "password, p",
			Value:       "",
			Usage:       " identity password",
			Destination: &globals.password,
		},
		cli.StringFlag{
			Name:        "use-agent, u",
			Value:       "",
			Usage:       " executable program that returns the password",
			Destination: &globals.agent,
		},
		cli.BoolFlag{
			Name:        "zero-agent-cache, z",
			Usage:       " force re-entry of agent password",
			Destination: &globals.clearCache,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "generate",
			Usage:     "generate key pair, will not store in config file",
			ArgsUsage: "\n   (* = required)",
			Flags:     []cli.Flag{},
			Action: func(c *cli.Context) {
				runGenerate(c, globals)
			},
		},
		{
			Name:      "setup",
			Usage:     "Initialise bitmark-cli configuration",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "network, n",
					Value: "",
					Usage: " bitmark|testing|local. Connect to which bitmark network [testing]",
				},
				cli.StringFlag{
					Name:  "connect, x",
					Value: "",
					Usage: "*bitmarkd host/IP and port, HOST:PORT",
				},
				cli.StringFlag{
					Name:  "description, d",
					Value: "",
					Usage: "*identity description",
				},
				cli.StringFlag{
					Name:  "privateKey, k",
					Value: "",
					Usage: " using existing privateKey",
				},
			},
			Action: func(c *cli.Context) {
				runSetup(c, globals)
			},
		},
		{
			Name:      "add",
			Usage:     "add identity to config file, set it as default",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "description, d",
					Value: "",
					Usage: "*identity descriptiont",
				},
				cli.StringFlag{
					Name:  "privateKey, k",
					Value: "",
					Usage: " using existing privateKey",
				},
			},
			Action: func(c *cli.Context) {
				runAdd(c, globals)
			},
		},
		{
			Name:      "create",
			Usage:     "create a new bitmark",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "asset, a",
					Value: "",
					Usage: "*asset name",
				},
				cli.StringFlag{
					Name:  "metadata, m",
					Value: "",
					Usage: "*asset metadata",
				},
				cli.StringFlag{
					Name:  "fingerprint, f",
					Value: "",
					Usage: "*asset fingerprint",
				},
				cli.StringFlag{
					Name:  "quantity, q",
					Value: "",
					Usage: " quantity to create [1]",
				},
			},
			Action: func(c *cli.Context) {
				runCreate(c, globals)
			},
		},
		{
			Name:      "transfer",
			Usage:     "transfer bitmark",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "txid, t",
					Value: "",
					Usage: "*transaction id to transfer",
				},
				cli.StringFlag{
					Name:  "receiver, r",
					Value: "",
					Usage: "*identity name to receive the bitmark",
				},
			},
			Action: func(c *cli.Context) {
				runTransfer(c, globals)
			},
		},
		{
			Name:      "receipt",
			Usage:     "receipt payment transaction id",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "payid, p",
					Value: "",
					Usage: "*payment id from a transfer",
				},
				cli.StringFlag{
					Name:  "receipt, r",
					Value: "",
					Usage: "*hexadecimal transaction id from currency transfer",
				},
			},
			Action: func(c *cli.Context) {
				runReceipt(c, globals)
			},
		},
		{
			Name:      "provenance",
			Usage:     "provenance bitmark",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "txid, t",
					Value: "",
					Usage: "*transaction id to list provenance",
				},
				cli.StringFlag{
					Name:  "count, c",
					Value: "",
					Usage: " maximum records to output [20]",
				},
			},
			Action: func(c *cli.Context) {
				runProvenance(c, globals)
			},
		}, {
			Name:  "info",
			Usage: "display bitmark-cli status",
			Action: func(c *cli.Context) {
				runInfo(c, globals)
			},
		},
		{
			Name:  "bitmarkInfo",
			Usage: "display bitmarkd status",
			Action: func(c *cli.Context) {
				runBitmarkInfo(c, globals)
			},
		},
		{
			Name:  "keypair",
			Usage: "get default identity's raw key pair",
			Action: func(c *cli.Context) {
				getDefaultRawKeyPair(c, globals)
			},
		},
		{
			Name:  "password",
			Usage: "change default identity's passwordr",
			Action: func(c *cli.Context) {
				changePassword(c, globals)
			},
		},
		{
			Name:  "version",
			Usage: "display bitmark-cli version",
			Action: func(c *cli.Context) {
				fmt.Println(Version)
			},
		},
	}

	app.Run(os.Args)
}

func runGenerate(c *cli.Context, globals globalFlags) {
	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	if err := makeRawKeyPair("bitmark" != configData.Network); nil != err {
		exitwithstatus.Message("Error: %s", err)
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

	if !addIdentity(configData, name, description, privateKey, globals.password) {
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

	if !addIdentity(configData, name, description, privateKey, globals.password) {
		exitwithstatus.Message("Error: add failed")
	}
	err = configuration.Save(configFile, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}
}

func runCreate(c *cli.Context, globals globalFlags) {

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

	err = issue(bitmarkRpcConfig, assetConfig, verbose)
	if nil != err {
		exitwithstatus.Message("Issue error: %s", err)
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
		if len(newPublicKey) != len(newOwnerKeyPair.PublicKey) {
			exitwithstatus.Message("hex public key must be %d bytes", publicKeySize)
		}
		copy(newOwnerKeyPair.PublicKey[:], newPublicKey)
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

	transferConfig := transferData{
		owner:    ownerKeyPair,
		newOwner: newOwnerKeyPair,
		txId:     txId,
	}

	err = transfer(bitmarkRpcConfig, transferConfig, verbose)
	if nil != err {
		exitwithstatus.Message("Transfer error: %s", err)
	}
}

func runReceipt(c *cli.Context, globals globalFlags) {

	configData, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	payId, err := checkPayId(c.String("payid"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	receiptId, err := checkReceipt(c.String("receipt"))
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("payid: %s\n", payId)
		fmt.Printf("receipt: %s\n", receiptId)
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configData.Connect,
		network:  configData.Network,
	}

	receiptConfig := receiptData{
		payId:   payId,
		receipt: receiptId,
	}

	err = receipt(bitmarkRpcConfig, receiptConfig, verbose)
	if nil != err {
		exitwithstatus.Message("Receipt error: %s", err)
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

func runInfo(c *cli.Context, globals globalFlags) {

	infoConfig, err := configuration.GetInfoConfiguration(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %s", err)
	}

	output, err := json.MarshalIndent(infoConfig, "", "  ")
	if nil != err {
		exitwithstatus.Message("Error: Marshal config failed: %s", err)
	}

	fmt.Println(string(output))
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

	if !bitmarkInfo(bitmarkRpcConfig, verbose) {
		exitwithstatus.Message("Error: Get info failed")
	}
}

func addIdentity(configs *configuration.Configuration, name string, description string, privateKeyStr string, password string) bool {

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

	publicKey, encryptPrivateKey, privateKeyConfig, err := makeKeyPair(privateKeyStr, password)
	if nil != err {
		fmt.Printf("error generating server key pair: %s\n", err)
		return false
	}

	identity := configuration.IdentityType{
		Name:               name,
		Description:        description,
		Public_key:         publicKey,
		Private_key:        encryptPrivateKey,
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
	err = doIssues(client, rpcConfig.network, issueConfig, verbose)
	if nil != err {
		return err
	}

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

	err = doTransfer(client, rpcConfig.network, transferConfig, verbose)
	if nil != err {
		return err
	}
	return nil
}

func receipt(rpcConfig bitmarkRPC, receiptConfig receiptData, verbose bool) error {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		return err
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	err = doReceipt(client, rpcConfig.network, receiptConfig, verbose)
	if nil != err {
		return err
	}
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

	err = doProvenance(client, rpcConfig.network, provenanceConfig, verbose)
	if nil != err {
		return err
	}
	return nil
}

func bitmarkInfo(rpcConfig bitmarkRPC, verbose bool) bool {
	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		fmt.Printf("Error: %s\n", err)
		return false
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	err = getBitmarkInfo(client, verbose)
	if nil != err {
		fmt.Printf("Error: %s\n", err)
		return false
	}
	return true
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
			Seed:       "?",
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

	publicKey, encryptPrivateKey, privateKeyConfig, err := makeKeyPair(hex.EncodeToString(keyPair.PrivateKey[:]), newPassword)
	if nil != err {
		exitwithstatus.Message("make key pair error: %s", err)
	}
	if publicKey != identity.Public_key {
		exitwithstatus.Message("public key was modified", err)
	}
	identity.Private_key = encryptPrivateKey
	identity.Private_key_config = *privateKeyConfig

	err = configuration.Save(configFile, configData)
	if nil != err {
		exitwithstatus.Message("Error: %s", err)
	}
}
