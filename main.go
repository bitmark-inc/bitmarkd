// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/go-programs/bitmark-cli/configuration"
	"github.com/bitmark-inc/go-programs/bitmark-cli/fault"
	"github.com/bitmark-inc/go-programs/bitmark-cli/templates"
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
			Usage:       "*bitmark-cli config folder",
			Destination: &globals.config,
		},
		cli.StringFlag{
			Name:        "identity, i",
			Value:       "",
			Usage:       " identity name [default identity]",
			Destination: &globals.identity,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "setup",
			Usage:     "Initialise bitmark-cli configuration",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "network, n",
					Value: "",
					Usage: " bitmark|testing. Connect to which bitmark network [testing]",
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
			},
			Action: func(c *cli.Context) {
				runSetup(c, globals)
			},
		},
		{
			Name:      "generate",
			Usage:     "new identity",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "description, d",
					Value: "",
					Usage: "*identity descriptiont",
				},
			},
			Action: func(c *cli.Context) {
				runGenerate(c, globals)
			},
		},
		{
			Name:      "issue",
			Usage:     "create and issue bitmark",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "asset, a",
					Value: "",
					Usage: "*asset name",
				},
				cli.StringFlag{
					Name:  "description, d",
					Value: "",
					Usage: "*asset description",
				},
				cli.StringFlag{
					Name:  "fingerprint, f",
					Value: "",
					Usage: "*asset fingerprint",
				},
				cli.StringFlag{
					Name:  "quantity, q",
					Value: "",
					Usage: " quantity to issue [1]",
				},
			},
			Action: func(c *cli.Context) {
				runIssue(c, globals)
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
					Usage: "*identity name to receive the transactoin",
				},
			},
			Action: func(c *cli.Context) {
				runTransfer(c, globals)
			},
		},
		{
			Name:  "info",
			Usage: "display bitmarkd status",
			Action: func(c *cli.Context) {
				runInfo(c, globals)
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

func runSetup(c *cli.Context, globals globalFlags) {

	configDir, err := checkConfigDir(globals.config)
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

	verbose := globals.verbose
	if verbose {
		fmt.Printf("config: %s\n", configDir)
		fmt.Printf("network: %s\n", network)
		fmt.Printf("connect: %s\n", connect)
		fmt.Printf("identity: %s\n", name)
		fmt.Printf("description: %s\n", description)
		fmt.Println()
	}

	// Create the folder if not existed
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

	if !(generateConfiguration(configDir, configData) &&
		generateIdentity(configDir, name, description)) {
		exitwithstatus.Message("Error: Setup failed\n")
	}
}

func runGenerate(c *cli.Context, globals globalFlags) {

	configDir, err := checkConfigDir(globals.config)
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
		fmt.Printf("config: %s\n", configDir)
		fmt.Printf("identity: %s\n", name)
		fmt.Printf("description: %s\n", description)
		fmt.Println()
	}

	if !generateIdentity(configDir, name, description) {
		exitwithstatus.Message("Error: generate failed\n")
	}
}

func runIssue(c *cli.Context, globals globalFlags) {

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

	description, err := checkAssetDescription(c.String("description"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	fingerprint, err := checkAssetFingerprint(c.String("fingerprint"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	quantity, err := checkAssetQuantity(c.String("quantity"))
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	verbose := globals.verbose
	if verbose {
		fmt.Printf("issuer: %v\n", issuer.Name)
		fmt.Printf("assetName: %v\n", assetName)
		fmt.Printf("description: %v\n", description)
		fmt.Printf("fingerprint: %v\n", fingerprint)
		fmt.Printf("quantity: %d\n", quantity)
	}

	// check password
	publicKey, privateKey, err := promptAndCheckPassword(issuer)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configuration.Connect,
		testNet:  true,
	}
	if configuration.Network != "testing" {
		bitmarkRpcConfig.testNet = false
	}

	registrant := keyPair{
		publicKey:  *publicKey,
		privateKey: *privateKey,
	}

	assetConfig := assetData{
		name:        assetName,
		description: description,
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

	to, err := checkTransferTo(c.String("receiver"), configuration.Identities)
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
		fmt.Printf("receiver: %s\n", to.Name)
		fmt.Printf("sender: %s\n", from.Name)
	}

	// check owner password
	publicKey, privateKey, err := promptAndCheckPassword(from)
	if nil != err {
		exitwithstatus.Message("Error: %s\n", err)
	}

	ownerKeyPair := keyPair{
		publicKey:  *publicKey,
		privateKey: *privateKey,
	}

	tmpPublicKey, err := hex.DecodeString(to.Public_key)
	if nil != err {
		fmt.Printf("Decode to public key error\n")
		exitwithstatus.Message("Error: %s\n", err)
	}

	newOwnerKeyPair := keyPair{}
	copy(newOwnerKeyPair.publicKey[:], tmpPublicKey[:])

	// TODO: deal with IPv6?
	bitmarkRpcConfig := bitmarkRPC{
		hostPort: configuration.Connect,
		testNet:  true,
	}
	if configuration.Network != "testing" {
		bitmarkRpcConfig.testNet = false
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

func runInfo(c *cli.Context, globals globalFlags) {

	configuration, err := checkAndGetConfig(globals.config)
	if nil != err {
		exitwithstatus.Message("Error: Get configuration failed: %v\n", err)
	}

	verbose := globals.verbose
	if !info(configuration.Connect, verbose) {
		exitwithstatus.Message("Error: Get info failed\n")
	}
}

func generateConfiguration(configDir string, configData configuration.Configuration) bool {

	configFile, err := configuration.GetConfigPath(configDir)
	if nil != err {
		fmt.Printf("Get config file failed: %v\n", err)
		return false
	}

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

func generateIdentity(configDir string, name string, description string) bool {

	configFile, err := configuration.GetConfigPath(configDir)
	if nil != err {
		fmt.Printf("Get config file failed: %v\n", err)
		return false
	}

	if !ensureFileExists(configFile) {
		fmt.Printf("Error: %v: %s\n", fault.ErrNotFoundConfigFile, configFile)
		return false
	} else {
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
	}

	// prompt password and pwd confirm for private key encryption
	password, err := promptPasswordReader()
	if nil != err {
		fmt.Printf("input password fail: %s\n", err)
		return false
	}
	publicKey, encryptPrivateKey, privateKeyConfig, err := makeKeyPair(name, password)
	if nil != err {
		cleanPasswordMemory(&password)
		fmt.Printf("error generating server key pair: %v\n", err)
		return false
	}

	// rewrite password memory
	cleanPasswordMemory(&password)

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
	assetIndex, err := makeAsset(client, rpcConfig.testNet, assetConfig, verbose)
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
	err = doIssues(client, rpcConfig.testNet, issueConfig, verbose)
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

	err = doTransfer(client, rpcConfig.testNet, transferConfig, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	return true
}

func info(hostPort string, verbose bool) bool {
	conn, err := connect(hostPort)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	err = getInfo(client, verbose)
	if nil != err {
		fmt.Printf("Error: %v\n", err)
		return false
	}
	return true
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
