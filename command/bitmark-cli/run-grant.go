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

func runGrant(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	to, recipient, err := checkRecipient(c, "receiver", m.config)
	if nil != err {
		return err
	}

	shareId, err := checkTxId(c.String("share-id"))
	if nil != err {
		return err
	}

	quantity := c.Uint64("quantity")
	if quantity == 0 {
		return fmt.Errorf("invalid quantity: %d", quantity)
	}

	beforeBlock := c.Uint64("before-block")

	from, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "shareId: %s\n", shareId)
		fmt.Fprintf(m.e, "quantity: %d\n", quantity)
		fmt.Fprintf(m.e, "sender: %s\n", from)
		fmt.Fprintf(m.e, "receiver: %s\n", to)
		fmt.Fprintf(m.e, "beforeBlock: %d\n", beforeBlock)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	grantConfig := &rpccalls.GrantData{
		ShareId:     shareId,
		Quantity:    quantity,
		Owner:       owner,
		Recipient:   recipient,
		BeforeBlock: beforeBlock,
	}

	response, err := client.Grant(grantConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
