// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
)

func runTransfer(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	txId, err := checkTxId(c.String("txid"))
	if nil != err {
		return err
	}

	to, recipient, err := checkRecipient(c, "receiver", m.config)
	if nil != err {
		return err
	}

	from, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "txid: %s\n", txId)
		fmt.Fprintf(m.e, "receiver: %s\n", to)
		fmt.Fprintf(m.e, "sender: %s\n", from)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	transferConfig := &rpccalls.TransferData{
		Owner:    owner,
		NewOwner: recipient,
		TxId:     txId,
	}

	if c.Bool("unratified") {

		response, err := client.Transfer(transferConfig)
		if nil != err {
			return err
		}

		printJson(m.w, response)

	} else {
		response, err := client.SingleSignedTransfer(transferConfig)
		if nil != err {
			return err
		}

		printJson(m.w, response)
	}
	return nil
}
