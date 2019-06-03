// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
)

func runBalance(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	shareId := c.String("share-id")

	owner := c.String("owner")
	if "" == owner {
		owner = c.GlobalString("identity")
		if "" == owner {
			owner = m.config.DefaultIdentity
		}
	}

	count := c.Int("count")
	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	if m.verbose {
		fmt.Fprintf(m.e, "owner: %s\n", owner)
		fmt.Fprintf(m.e, "shareId: %s\n", shareId)
		fmt.Fprintf(m.e, "count: %d\n", count)
	}

	ownerKeyPair, err := encrypt.PublicKeyFromString(owner, m.config.Identities, m.config.TestNet)
	if nil != err {
		return err
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connect, m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	balanceConfig := &rpccalls.BalanceData{
		Owner:   ownerKeyPair,
		ShareId: shareId,
		Count:   count,
	}

	response, err := client.GetBalance(balanceConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
