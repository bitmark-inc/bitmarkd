// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
)

type metadata struct {
	file    string
	config  *configuration.Configuration
	save    bool
	testnet bool
	verbose bool
	e       io.Writer
	w       io.Writer
}

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version = "zero" // do not change this value

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
			Name:  "network, n",
			Value: "",
			Usage: " connect to bitmark `NETWORK` [bitmark|testing|local]",
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
			ArgsUsage: "\n   (* = required, + = select one)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "connect, c",
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
				cli.BoolFlag{
					Name:  "zero, z",
					Usage: " only try to issue the free zero nonce",
				},
				cli.IntFlag{
					Name:  "quantity, q",
					Value: 1,
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
				cli.BoolFlag{
					Name:  "unratified, u",
					Usage: " perform an unratified transfer (default is output single signed hex)",
				},
			},
			Action: runTransfer,
		},
		{
			Name:      "countersign",
			Usage:     "countersign a transaction using current identity",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "transaction, t",
					Value: "",
					Usage: "*sender signed transfer `HEX` code",
				},
			},
			Action: runCountersign,
		},
		{
			Name:      "blocktransfer",
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
					Usage: "*identity name to receive the block `ACCOUNT`",
				},
				cli.StringFlag{
					Name:  "bitcoin, b",
					Value: "",
					Usage: "*address receive the bitcoin payment `ACCOUNT`",
				},
				cli.StringFlag{
					Name:  "litecoin, l",
					Value: "",
					Usage: "*address to receive the litecoin payment `ACCOUNT`",
				},
			},
			Action: runBlockTransfer,
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
				cli.IntFlag{
					Name:  "count, c",
					Value: 20,
					Usage: " maximum records to output `COUNT`",
				},
			},
			Action: runProvenance,
		},
		{
			Name:      "owned",
			Usage:     "list bitmarks owned",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "owner, o",
					Value: "",
					Usage: " identity name `ACCOUNT` default is global identity",
				},
				cli.Uint64Flag{
					Name:  "start, s",
					Value: 0,
					Usage: " start point `COUNT`",
				},
				cli.IntFlag{
					Name:  "count, c",
					Value: 20,
					Usage: " maximum records to output `COUNT`",
				},
			},
			Action: runOwned,
		},
		{
			Name:      "share",
			Usage:     "convert a bitmark into a share",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "txid, t",
					Value: "",
					Usage: "*transaction id to convert `TXID`",
				},
				cli.IntFlag{
					Name:  "quantity, q",
					Value: 0,
					Usage: "*quantity to create `NUMBER`",
				},
			},
			Action: runShare,
		},
		{
			Name:      "grant",
			Usage:     "grant some shares of a bitmark to a receiver",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "receiver, r",
					Value: "",
					Usage: "*identity name to receive the block `ACCOUNT`",
				},
				cli.StringFlag{
					Name:  "share-id, s",
					Value: "",
					Usage: "*transaction id of share `SHARE_ID`",
				},
				cli.Uint64Flag{
					Name:  "quantity, q",
					Value: 1,
					Usage: " quantity to grant `NUMBER`",
				},
				cli.Uint64Flag{
					Name:  "before-block, b",
					Value: 0,
					Usage: " must confirm before this block `NUMBER`",
				},
			},
			Action: runGrant,
		},
		{
			Name:      "swap",
			Usage:     "swap some shares of a bitmark to a receiver",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "receiver, r",
					Value: "",
					Usage: "*identity name to receive the block `ACCOUNT`",
				},
				cli.StringFlag{
					Name:  "share-id-one, s",
					Value: "",
					Usage: "*transaction id of share one `SHARE_ID`",
				},
				cli.Uint64Flag{
					Name:  "quantity-one, q",
					Value: 1,
					Usage: " quantity of share one `NUMBER`",
				},
				cli.StringFlag{
					Name:  "share-id-two, S",
					Value: "",
					Usage: "*transaction id of share two `SHARE_ID`",
				},
				cli.Uint64Flag{
					Name:  "quantity-two, Q",
					Value: 1,
					Usage: " quantity of share two `NUMBER`",
				},
				cli.Uint64Flag{
					Name:  "before-block, b",
					Value: 0,
					Usage: " must confirm before this block `NUMBER`",
				},
			},
			Action: runSwap,
		},
		{
			Name:      "balance",
			Usage:     "display balance of some shares",
			ArgsUsage: "\n   (* = required)",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "owner, o",
					Value: "",
					Usage: " identity name `ACCOUNT` default is global identity",
				},
				cli.StringFlag{
					Name:  "share-id, s",
					Value: "",
					Usage: " starting from share `SHARE_ID`",
				},
				cli.IntFlag{
					Name:  "count, c",
					Value: 20,
					Usage: " maximum records to output `COUNT`",
				},
			},
			Action: runBalance,
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

		// only want one of these
		network := c.GlobalString("network")
		switch network {
		case "bitmark", "live":
			network = "bitmark"
		case "testing", "test":
			network = "testing"
		case "local", "regression":
			network = "local"
		default:
			return fmt.Errorf("network: %q can only be bitmark/testing/local", network)
		}

		p := os.Getenv("XDG_CONFIG_HOME")
		if "" == p {
			return fmt.Errorf("XDG_CONFIG_HOME environment is not set")
		}
		dir, err := checkFileExists(p)
		if nil != err {
			return err
		}
		if !dir {
			return fmt.Errorf("not a directory: %q", p)
		}
		file := path.Join(p, app.Name, network+"-"+app.Name+".json")

		if verbose {
			fmt.Fprintf(e, "file: %q\n", file)
		}

		if "setup" == command {
			// do not run setup if there is an existing configuration
			if _, err := checkFileExists(file); nil == err {
				return fmt.Errorf("not overwriting existing configuration: %q", file)
			}

			c.App.Metadata["config"] = &metadata{
				file:    file,
				save:    false,
				testnet: network != "bitmark",
				verbose: verbose,
				e:       e,
				w:       w,
			}

		} else {

			if verbose {
				fmt.Fprintf(e, "reading config file: %s\n", file)
			}

			configuration, err := configuration.GetConfiguration(file)
			if nil != err {
				return err
			}

			c.App.Metadata["config"] = &metadata{
				file:    file,
				config:  configuration,
				testnet: configuration.TestNet,
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
