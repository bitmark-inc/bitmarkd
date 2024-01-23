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

func runTransactionStatus(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	txId, err := checkTxId(c.String("txid"))
	if err != nil {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "txid: %s\n", txId)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if err != nil {
		return err
	}
	defer client.Close()

	statusConfig := &rpccalls.TransactionStatusData{
		TxId: txId,
	}

	response, err := client.GetTransactionStatus(statusConfig)
	if err != nil {
		return err
	}

	printJson(m.w, response)

	return nil
}
