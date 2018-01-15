// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/urfave/cli"
	"io"
	"os"
)

type metadata struct {
	file      string
	config    *configuration.Configuration
	save      bool
	testnet   bool
	verbose   bool
	variables map[string]string
	e         io.Writer
	w         io.Writer
}

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version string = "zero" // do not change this value

func main() {

	app := cli.NewApp()
	app.Name = "bitmark-cli"
	// app.Usage = ""
	app.Version = version
	app.HideVersion = true

	app.Writer = os.Stdout
	app.ErrWriter = os.Stderr

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "verbose, v",
			Usage: " verbose result",
		},
		cli.StringFlag{
			Name:  "config, c",
			Value: "",
			Usage: "bitmark-cli configuration `DIRECTORY`",
		},
		cli.StringFlag{
			Name:  "identity, i",
			Value: "",
			Usage: " identity `NAME` [default identity]",
		},
		cli.StringFlag{
			Name:  "password, p",
			Value: "",
			Usage: " identity `PASSWORD`",
		},
		cli.StringFlag{
			Name:  "use-agent, u",
			Value: "",
			Usage: " executable program that returns the password `EXE`",
		},
		cli.BoolFlag{
			Name:  "zero-agent-cache, z",
			Usage: " force re-entry of agent password",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "generate",
			Usage:     "generate key pair, will not store in config file",
			ArgsUsage: "\n   (* = required)",
			Flags:     []cli.Flag{},
			Action:    runGenerate,
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
			Action: runSetup,
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
			Action: runAdd,
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
			Action: runCreate,
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
			Action: runTransfer,
		},
		{
			Name:      "countersign",
			Usage:     "countersign transfer a bitmark to current account",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "transfer, t",
					Value: "",
					Usage: "*sender signed transfer `HEX` code",
				},
			},
			Action: runCountersign,
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
			Action: runProvenance,
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
			Action: runTransactionStatus,
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
			Action: runAccount,
		},
		{
			Name:   "info",
			Usage:  "display bitmark-cli status",
			Action: runInfo,
		},
		{
			Name:   "bitmarkInfo",
			Usage:  "display bitmarkd status",
			Action: runBitmarkInfo,
		},
		{
			Name:   "keypair",
			Usage:  "get default identity's raw key pair",
			Action: runKeyPair,
		},
		{
			Name:   "password",
			Usage:  "change default identity's password",
			Action: runChangePassword,
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
			Action: runFingerprint,
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
			Action: runSign,
		},
		{
			Name:  "version",
			Usage: "display bitmark-cli version",
			Action: func(c *cli.Context) error {
				fmt.Fprintf(c.App.Writer, "%s\n", version)
				return nil
			},
		},
	}

	// read the configuration
	app.Before = func(c *cli.Context) error {

		e := c.App.ErrWriter
		w := c.App.Writer
		verbose := c.GlobalBool("verbose")

		// to suppress reading config file if certain commands
		command := c.Args().Get(0)
		if "version" == command {
			return nil
		}

		file := c.GlobalString("config")
		if "" == file {
			return ErrRequiredConfigFile
		}

		// expand ${HOME} etc.
		file = os.ExpandEnv(file)

		// config file macros - currently empty
		variables := make(map[string]string)

		if "setup" == command {
			// do not run setup if there is an existing configuration
			if ensureFileExists(file) {
				return fmt.Errorf("not overwriting existing configuration: %q", file)
			}

			c.App.Metadata["config"] = &metadata{
				file:      file,
				save:      false,
				variables: variables,
				verbose:   verbose,
				e:         e,
				w:         w,
			}

		} else {

			if verbose {
				fmt.Fprintf(e, "reading config file: %s\n", file)
			}

			configuration, err := configuration.GetConfiguration(file, variables)
			if nil != err {
				return err
			}

			c.App.Metadata["config"] = &metadata{
				file:    file,
				config:  configuration,
				testnet: "bitmark" != configuration.Network,
				save:    false,
				verbose: verbose,
				e:       e,
				w:       w,
			}
		}

		return nil
	}

	// update the configuration if required
	app.After = func(c *cli.Context) error {
		e := c.App.ErrWriter
		m, ok := c.App.Metadata["config"].(*metadata)
		if !ok {
			return nil
		}
		if m.save {
			if c.GlobalBool("verbose") {
				fmt.Fprintf(e, "updating config file: %s\n", m.file)
			}
			err := configuration.Save(m.file, m.config)
			if nil != err {
				return err
			}
		}
		return nil
	}

	err := app.Run(os.Args)
	if nil != err {
		fmt.Fprintf(app.ErrWriter, "terminated with error: %s\n", err)
		os.Exit(1)
	}
}
