// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmark-cli/fault"
	"github.com/bitmark-inc/bitmark-cli/templates"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"net/rpc/jsonrpc"
	"os"
	"strings"
	"text/template"
)

type globalFlags struct {
	verbose  bool
	config   string
	identity string
	password string
}

func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	globals := globalFlags{}

	app := cli.NewApp()
	app.Name = "bitmark-cli"
	// app.Usage = ""
	app.Version = Version()
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
			Usage:       "bitmark-cli configuration dir",
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
					Usage: "using existing privateKey",
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
			Name:  "version",
			Usage: "display bitmark-cli version",
			Action: func(c *cli.Context) {
				fmt.Println(Version())
			},
		},
	}

	app.Run(os.Args)
}

func runGenerate(c *cli.Context, globals globalFlags) {
	if err := makeRawKeyPair(); nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}
}

func runSetup(c *cli.Context, globals globalFlags) {
	configFile, err := checkConfigFile(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	name, err := checkName(globals.identity)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	network := checkNetwork(c.String("network"))

	connect, err := checkConnect(c.String("connect"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	description, err := checkDescription(c.String("description"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	privateKey := c.String("privateKey")

	verbose := globals.verbose
	if verbose {
		fmt.Printf("config: %s\n", configFile)
		fmt.Printf("network: %s\n", network)
		fmt.Printf("connect: %s\n", connect)
		fmt.Printf("identity: %s\n", name)
		fmt.Printf("description: %s\n", description)
		fmt.Println()
	}

	// Create the folder if not existing
	folderIndex := strings.LastIndex(configFile, "/")
	configDir := configFile[:folderIndex]
	if !ensureFileExists(configDir) {
		if err := os.MkdirAll(configDir, 0755); nil != err {
			exitwithstatus.Message("Error: %v\n", err)
		}
	}

	configData := configuration.Configuration{
		Default_identity: name,
		Network:          network,
		Connect:          connect,
		Identities:       make([]configuration.IdentityType, 0),
	}

	if !(generateConfiguration(configFile, configData) &&
		generateIdentity(configFile, name, description, privateKey, globals.password)) {
		exitwithstatus.Message("Error: Setup failed\n")
	}
}

func runAdd(c *cli.Context, globals globalFlags) {

	configFile, err := checkConfigFile(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	name, err := checkName(globals.identity)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	description, err := checkDescription(c.String("description"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("config: %s\n", configFile)
		fmt.Printf("identity: %s\n", name)
		fmt.Printf("description: %s\n", description)
		fmt.Println()
	}

	if !generateIdentity(configFile, name, description, "", globals.password) {
		exitwithstatus.Message("Error: add failed\n")
	}
}

func runCreate(c *cli.Context, globals globalFlags) {

	configuration, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	issuer, err := checkIdentity(globals.identity, configuration)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	assetName, err := checkAssetName(c.String("asset"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	fingerprint, err := checkAssetFingerprint(c.String("fingerprint"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	metadata, err := checkAssetMetadata(c.String("metadata"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	quantity, err := checkAssetQuantity(c.String("quantity"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
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
	if "" == globals.password {
		registrant, err = promptAndCheckPassword(issuer)
		if nil != err {
			exitwithstatus.Message("Error: %s\n", err)
		}
	} else {
		registrant, err = verifyPassword(globals.password, issuer)
		if nil != err {
			exitwithstatus.Message("Error: %s\n", err)
		}
	}
	// just in case some internal breakage
	if nil == registrant {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configuration.Connect,
		network:  configuration.Network,
	}

	assetConfig := assetData{
		name:        assetName,
		metadata:    metadata,
		quantity:    quantity,
		fingerprint: fingerprint,
		registrant:  registrant,
	}

	if !issue(bitmarkRpcConfig, assetConfig, verbose) {
		exitwithstatus.Message("Error: issue failed\n")
	}
}

func runTransfer(c *cli.Context, globals globalFlags) {

	configuration, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	txId, err := checkTransferTxId(c.String("txid"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	to, err := checkTransferTo(c.String("receiver"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	from, err := checkTransferFrom(globals.identity, configuration)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("txid: %s\n", txId)
		fmt.Printf("receiver: %s\n", to)
		fmt.Printf("sender: %s\n", from.Name)
	}

	var ownerKeyPair *KeyPair
	// check owner password
	if "" == globals.password {
		ownerKeyPair, err = promptAndCheckPassword(from)
		if nil != err {
			exitwithstatus.Message("Error: %s\n", err)
		}
	} else {
		ownerKeyPair, err = verifyPassword(globals.password, from)
		if nil != err {
			exitwithstatus.Message("Error: %s\n", err)
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

		newOwnerKeyPair, err = publicKeyFromIdentity(to, configuration.Identities)
		if nil != err {
			exitwithstatus.Message("receiver identity error: %s\n", err)
		}
	} else {
		newOwnerKeyPair = &KeyPair{}
		if len(newPublicKey) != len(newOwnerKeyPair.PublicKey) {
			exitwithstatus.Message("hex public key must be 32 bytes")
		}
		copy(newOwnerKeyPair.PublicKey[:], newPublicKey)
	}
	// just in case some internal breakage
	if nil == newOwnerKeyPair {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configuration.Connect,
		network:  configuration.Network,
	}

	transferConfig := transferData{
		owner:    ownerKeyPair,
		newOwner: newOwnerKeyPair,
		txId:     txId,
	}

	if !transfer(bitmarkRpcConfig, transferConfig, verbose) {
		exitwithstatus.Message("Error: Transfer failed\n")
	}
}

func runReceipt(c *cli.Context, globals globalFlags) {

	configuration, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	payId, err := checkPayId(c.String("payid"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	receiptId, err := checkReceipt(c.String("receipt"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("payid: %s\n", payId)
		fmt.Printf("receipt: %s\n", receiptId)
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configuration.Connect,
		network:  configuration.Network,
	}

	receiptConfig := receiptData{
		payId:   payId,
		receipt: receiptId,
	}

	if !receipt(bitmarkRpcConfig, receiptConfig, verbose) {
		exitwithstatus.Message("Error: Receipt failed\n")
	}
}

func runProvenance(c *cli.Context, globals globalFlags) {

	configuration, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	txId, err := checkTransferTxId(c.String("txid"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	count, err := checkRecordCount(c.String("count"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("txid: %s\n", txId)
		fmt.Printf("count: %d\n", count)
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configuration.Connect,
		network:  configuration.Network,
	}

	provenanceConfig := provenanceData{
		txId:  txId,
		count: count,
	}

	if !provenance(bitmarkRpcConfig, provenanceConfig, verbose) {
		exitwithstatus.Message("Error: Provenance failed\n")
	}
}

func runInfo(c *cli.Context, globals globalFlags) {

	infoConfig, err := configuration.GetInfoConfiguration(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	output, err := json.MarshalIndent(infoConfig, "", "  ")
	if nil != err {
		exitwithstatus.Message("Error: Marshal config failed: %v\n", err)
	}

	fmt.Println(string(output))
}

func runBitmarkInfo(c *cli.Context, globals globalFlags) {

	configuration, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	verbose := globals.verbose

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configuration.Connect,
		network:  configuration.Network,
	}

	if !bitmarkInfo(bitmarkRpcConfig, verbose) {
		exitwithstatus.Message("Error: Get info failed\n")
	}
}

func generateConfiguration(configFile string, configData configuration.Configuration) bool {

	// Check if file exist
	if !ensureFileExists(configFile) {
		file, error := os.Create(configFile)
		if nil != error {
			fmt.Printf("Create file fail: %s\n", error)
			return false
		}

		confTemp := template.Must(template.New("config").Parse(templates.ConfigurationTemplate))
		error = confTemp.Execute(file, configData)
		if nil != error {
			fmt.Printf("Init Config file fail: %s\n", error)
		}
	} else {
		fmt.Printf("%s exists\n", configFile)
		return false
	}

	return true
}

func generateIdentity(configFile string, name string, description string, privateKeyStr string, password string) bool {

	if !ensureFileExists(configFile) {
		fmt.Printf("Error: %v: %s\n", fault.ErrNotFoundConfigFile, configFile)
		return false
	}

	configs, err := configuration.GetConfiguration(configFile)
	if nil != err {
		fmt.Printf("configuration fail: %s\n", err)
		return false
	}

	for _, identity := range configs.Identities {
		if name == identity.Name {
			fmt.Printf("identity exists. Name: %s\n", name)
			return false
		}
	}

	if "" == password {
		// prompt password and pwd confirm for private key encryption
		password, err = promptPasswordReader()
		if nil != err {
			fmt.Printf("input password fail: %s\n", err)
			return false
		}
	}

	publicKey, encryptPrivateKey, privateKeyConfig, err := makeKeyPair(privateKeyStr, password)
	if nil != err {
		fmt.Printf("error generating server key pair: %v\n", err)
		return false
	}

	identity := configuration.IdentityType{
		Name:               name,
		Description:        description,
		Public_key:         publicKey,
		Private_key:        encryptPrivateKey,
		Private_key_config: *privateKeyConfig,
	}
	if !writeIdentityToFile(identity, configFile) {
		fmt.Printf("Write identity to file failed\n: %v", identity)
		return false
	}

	return true
}

func issue(rpcConfig bitmarkRPC, assetConfig assetData, verbose bool) bool {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	// make asset
	assetIndex, err := makeAsset(client, rpcConfig.network, assetConfig, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}

	// make Issues
	issueConfig := issueData{
		issuer:     assetConfig.registrant,
		assetIndex: assetIndex,
		quantity:   assetConfig.quantity,
	}
	err = doIssues(client, rpcConfig.network, issueConfig, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}

	return true
}

func transfer(rpcConfig bitmarkRPC, transferConfig transferData, verbose bool) bool {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	err = doTransfer(client, rpcConfig.network, transferConfig, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	return true
}

func receipt(rpcConfig bitmarkRPC, receiptConfig receiptData, verbose bool) bool {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	err = doReceipt(client, rpcConfig.network, receiptConfig, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	return true
}

func provenance(rpcConfig bitmarkRPC, provenanceConfig provenanceData, verbose bool) bool {

	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	err = doProvenance(client, rpcConfig.network, provenanceConfig, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	return true
}

func bitmarkInfo(rpcConfig bitmarkRPC, verbose bool) bool {
	conn, err := connect(rpcConfig.hostPort)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	err = getBitmarkInfo(client, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	return true
}

func getDefaultRawKeyPair(c *cli.Context, globals globalFlags) {
	configuration, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	identity, err := checkTransferFrom(globals.identity, configuration)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	var keyPair *KeyPair

	// check owner password
	if "" == globals.password {
		keyPair, err = promptAndCheckPassword(identity)
		if nil != err {
			exitwithstatus.Message("Error: %s\n", err)
		}
	} else {
		keyPair, err = verifyPassword(globals.password, identity)
		if nil != err {
			exitwithstatus.Message("Error: %s\n", err)
		}
	}
	//just in case some internal breakage
	if nil == keyPair {
		exitwithstatus.Message("internal error: nil keypair returned")
	}

	rawKeyPair := RawKeyPair{
		PublicKey:  hex.EncodeToString(keyPair.PublicKey[:]),
		PrivateKey: hex.EncodeToString(keyPair.PrivateKey[:]),
	}
	if b, err := json.MarshalIndent(rawKeyPair, "", "  "); nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	} else {
		fmt.Printf("%s\n", b)
	}
}

func writeIdentityToFile(identity configuration.IdentityType, configFile string) bool {

	identityTemp := template.Must(template.New("identity").Parse(templates.IdentityTemplate))
	identityBuffer := new(bytes.Buffer)
	error := identityTemp.Execute(identityBuffer, identity)
	if nil != error {
		fmt.Printf("Generate identity file fail: %s\n", error)
		return false
	}

	// write identity under config identites
	input, error := ioutil.ReadFile(configFile)
	if nil != error {
		fmt.Printf("Read config file fail: %s\n", error)
		return false
	}

	lines := strings.Split(string(input), "\n")
	for i, line := range lines {
		if strings.Contains(line, "identities = [") {
			addIdentity := "identities = [" + identityBuffer.String()
			if strings.Contains(line, "]") { // empty identities
				addIdentity = addIdentity + "]"
			}
			lines[i] = addIdentity
		}
	}
	output := strings.Join(lines, "\n")
	error = ioutil.WriteFile(configFile, []byte(output), 0644)
	if nil != error {
		fmt.Printf("Write config file fail: %s\n", error)
		return false
	}

	return true
}
