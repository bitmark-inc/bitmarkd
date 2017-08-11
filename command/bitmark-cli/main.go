// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/codegangsta/cli"
	"os"
)

type globalFlags struct {
	verbose    bool
	config     string
	identity   string
	password   string
	agent      string
	clearCache bool
	variables  map[string]string
}

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version string = "zero" // do not change this value

func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	globals := globalFlags{}

	app := cli.NewApp()
	app.Name = "bitmark-cli"
	// app.Usage = ""
	app.Version = version
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
			Usage:       "bitmark-cli configuration `DIRECTORY`",
			Destination: &globals.config,
		},
		cli.StringFlag{
			Name:        "identity, i",
			Value:       "",
			Usage:       " identity `NAME` [default identity]",
			Destination: &globals.identity,
		},
		cli.StringFlag{
			Name:        "password, p",
			Value:       "",
			Usage:       " identity `PASSWORD`",
			Destination: &globals.password,
		},
		cli.StringFlag{
			Name:        "use-agent, u",
			Value:       "",
			Usage:       " executable program that returns the password `EXE`",
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
					Value: "testing",
					Usage: " bitmark|testing|local. Connect to bitmark `NETWORK`",
				},
				cli.StringFlag{
					Name:  "connect, x",
					Value: "",
					Usage: "*bitmarkd host/IP and port, `HOST:PORT`",
				},
				cli.StringFlag{
					Name:  "description, d",
					Value: "",
					Usage: "*identity description `STRING`",
				},
				cli.StringFlag{
					Name:  "privateKey, k",
					Value: "",
					Usage: " using existing privateKey/seed `KEY`",
				},
			},
			Action: func(c *cli.Context) {
				runSetup(c, globals)
			},
		},
		{
			Name:      "add",
			Usage:     "add a new identity to config file, set it as default",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "description, d",
					Value: "",
					Usage: "*identity description `STRING`",
				},
				cli.StringFlag{
					Name:  "privateKey, k",
					Value: "",
					Usage: " using existing privateKey/seed `KEY`",
				},
			},
			Action: func(c *cli.Context) {
				runAdd(c, globals)
			},
		},
		{
			Name:      "batch",
			Usage:     "create and transfer bitmarks to new accounts",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "asset, a",
					Value: "",
					Usage: "*asset name `STRING`",
				},
				cli.StringFlag{
					Name:  "metadata, m",
					Value: "",
					Usage: "*asset metadata `META`",
				},
				cli.StringFlag{
					Name:  "fingerprint, f",
					Value: "",
					Usage: "*asset fingerprint `STRING`",
				},
				cli.StringFlag{
					Name:  "quantity, q",
					Value: "1",
					Usage: " quantity to create `COUNT`",
				},
				cli.BoolFlag{
					Name:  "transfer, t",
					Usage: " to create quantity new accounts and transfer a bitmark to each one",
				},
				cli.StringFlag{
					Name:  "output, o",
					Value: "",
					Usage: " store final output in `FILE`",
				},
			},
			Action: func(c *cli.Context) {
				runCreate(c, globals, true)
			},
		},
		{
			Name:      "create",
			Usage:     "create one or more new bitmarks",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "asset, a",
					Value: "",
					Usage: "*asset name `STRING`",
				},
				cli.StringFlag{
					Name:  "metadata, m",
					Value: "",
					Usage: "*asset metadata `META`",
				},
				cli.StringFlag{
					Name:  "fingerprint, f",
					Value: "",
					Usage: "*asset fingerprint `STRING`",
				},
				cli.StringFlag{
					Name:  "quantity, q",
					Value: "1",
					Usage: " quantity to create `COUNT`",
				},
			},
			Action: func(c *cli.Context) {
				runCreate(c, globals, false)
			},
		},
		{
			Name:      "transfer",
			Usage:     "transfer a bitmark to another account",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "txid, t",
					Value: "",
					Usage: "*transaction id to transfer `TXID`",
				},
				cli.StringFlag{
					Name:  "receiver, r",
					Value: "",
					Usage: "*identity name to receive the bitmark `ACCOUNT`",
				},
			},
			Action: func(c *cli.Context) {
				runTransfer(c, globals)
			},
		},
		{
			Name:      "provenance",
			Usage:     "list provenance of a bitmark",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "txid, t",
					Value: "",
					Usage: "*transaction id to list provenance `TXID`",
				},
				cli.StringFlag{
					Name:  "count, c",
					Value: "20",
					Usage: " maximum records to output `COUNT`",
				},
			},
			Action: func(c *cli.Context) {
				runProvenance(c, globals)
			},
		},
		{
			Name:      "status",
			Usage:     "display the status of a bitmark",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "txid, t",
					Value: "",
					Usage: "*transaction id to check status `TXID`",
				},
			},
			Action: func(c *cli.Context) {
				runTransactionStatus(c, globals)
			},
		},
		{
			Name:      "account",
			Usage:     "display account from a public key",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "publickey, p",
					Value: "",
					Usage: "*hex public `KEY`",
				},
			},
			Action: func(c *cli.Context) {
				runPublicKeyDisplay(c, globals)
			},
		},
		{
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
			Usage: "change default identity's password",
			Action: func(c *cli.Context) {
				changePassword(c, globals)
			},
		},
		{
			Name:      "fingerprint",
			Usage:     "fingerprint a file (compatible to desktop app)",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Value: "",
					Usage: " `FILE` of data to fingerprint",
				},
			},
			Action: func(c *cli.Context) {
				runFingerprint(c, globals)
			},
		},
		{
			Name:      "sign",
			Usage:     "sign file",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "file, f",
					Value: "",
					Usage: " `FILE` of data to sign",
				},
			},
			Action: func(c *cli.Context) {
				runSign(c, globals)
			},
		},
		{
			Name:  "version",
			Usage: "display bitmark-cli version",
			Action: func(c *cli.Context) {
				fmt.Println(version)
			},
		},
	}

	app.Run(os.Args)
}
