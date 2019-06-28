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

func runCountersign(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	hex, err := checkTransferTx(c.String("transaction"))
	if nil != err {
		return err
	}

	// this command is run by the receiver so from is used to get

	to, newOwner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "tx: %s\n", hex)
		fmt.Fprintf(m.e, "receiver: %s\n", to)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[0], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	countersignConfig := &rpccalls.CountersignData{
		Transaction: hex,
		NewOwner:    newOwner,
	}

	response, err := client.Countersign(countersignConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
