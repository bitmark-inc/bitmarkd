// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/bitmark-inc/bitmarkd/currency"
)

func runBlockTransfer(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	txId, err := checkTxId(c.String("txid"))
	if nil != err {
		return err
	}

	to, newOwner, err := checkRecipient(c, "receiver", m.config)
	if nil != err {
		return err
	}

	bitcoinAddress, err := checkCoinAddress(currency.Bitcoin, c.String("bitcoin"), m.testnet)
	if nil != err {
		return err
	}
	litecoinAddress, err := checkCoinAddress(currency.Litecoin, c.String("litecoin"), m.testnet)
	if nil != err {
		return err
	}

	from, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if nil != err {
		return err
	}

	payments := currency.Map{
		currency.Bitcoin:  bitcoinAddress,
		currency.Litecoin: litecoinAddress,
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

	transferConfig := &rpccalls.BlockTransferData{
		Owner:    owner,
		NewOwner: newOwner,
		Payments: payments,
		TxId:     txId,
	}

	response, err := client.SingleSignedBlockTransfer(transferConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
