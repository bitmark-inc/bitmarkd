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
					Usage: " using existing privateKey/seed",
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
					Usage: "*identity description",
				},
				cli.StringFlag{
					Name:  "privateKey, k",
					Value: "",
					Usage: " using existing privateKey/seed",
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
				cli.StringFlag{
					Name:  "output, o",
					Value: "",
					Usage: " file to store final output",
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
			Name:      "provenance",
			Usage:     "list provenance of a bitmark",
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
		},
		{
			Name:      "status",
			Usage:     "display the status of a bitmark",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "txid, t",
					Value: "",
					Usage: "*transaction id to check status",
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
					Usage: "*hex public key",
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
					Usage: " file of data to fingerprint",
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
					Usage: " file of data to sign",
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
				fmt.Println(Version)
			},
		},
	}

	app.Run(os.Args)
}
