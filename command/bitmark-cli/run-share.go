// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
)

func runShare(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	txId, err := checkTxId(c.String("txid"))
	if err != nil {
		return err
	}

	quantity := c.Int("quantity")
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity: %d", quantity)
	}

	from, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if err != nil {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "from: %s\n", from)
		fmt.Fprintf(m.e, "txid: %s\n", txId)
		fmt.Fprintf(m.e, "quantity: %d\n", quantity)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if err != nil {
		return err
	}
	defer client.Close()

	// make Share
	shareConfig := &rpccalls.ShareData{
		Owner:    owner,
		TxId:     txId,
		Quantity: uint64(quantity),
	}

	response, err := client.Share(shareConfig)
	if err != nil {
		return err
	}

	printJson(m.w, response)
	return nil
}
