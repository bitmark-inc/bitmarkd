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

func runSwap(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	to, recipient, err := checkRecipient(c, "receiver", m.config)
	if err != nil {
		return err
	}

	shareIdOne, err := checkTxId(c.String("share-id-one"))
	if err != nil {
		return err
	}

	quantityOne := c.Uint64("quantity-one")
	if quantityOne == 0 {
		return fmt.Errorf("invalid quantity-one: %d", quantityOne)
	}

	shareIdTwo, err := checkTxId(c.String("share-id-two"))
	if err != nil {
		return err
	}

	quantityTwo := c.Uint64("quantity-two")
	if quantityTwo == 0 {
		return fmt.Errorf("invalid quantity-two: %d", quantityTwo)
	}

	beforeBlock := c.Uint64("before-block")

	from, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if err != nil {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "shareIdOne: %s\n", shareIdOne)
		fmt.Fprintf(m.e, "quantityOne: %d\n", quantityOne)
		fmt.Fprintf(m.e, "ownerOne: %s\n", from)
		fmt.Fprintf(m.e, "shareIdTwo: %s\n", shareIdTwo)
		fmt.Fprintf(m.e, "quantityTwo: %d\n", quantityTwo)
		fmt.Fprintf(m.e, "ownerTwo: %s\n", to)
		fmt.Fprintf(m.e, "beforeBlock: %d\n", beforeBlock)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if err != nil {
		return err
	}
	defer client.Close()

	swapConfig := &rpccalls.SwapData{
		ShareIdOne:  shareIdOne,
		QuantityOne: quantityOne,
		OwnerOne:    owner,
		ShareIdTwo:  shareIdTwo,
		QuantityTwo: quantityTwo,
		OwnerTwo:    recipient,
		BeforeBlock: beforeBlock,
	}

	response, err := client.Swap(swapConfig)
	if err != nil {
		return err
	}

	printJson(m.w, response)

	return nil
}
