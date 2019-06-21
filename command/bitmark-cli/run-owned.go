// SPDX-License-Identifier: ISC
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

func runOwned(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	owner := c.String("owner")
	if "" == owner {
		owner = c.GlobalString("identity")
		if "" == owner {
			owner = m.config.DefaultIdentity
		}
	}

	start := c.Uint64("start")

	count := c.Int("count")
	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	if m.verbose {
		fmt.Fprintf(m.e, "owner: %s\n", owner)
		fmt.Fprintf(m.e, "start: %d\n", start)
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

	ownedConfig := &rpccalls.OwnedData{
		Owner: ownerKeyPair,
		Start: start,
		Count: count,
	}

	response, err := client.GetOwned(ownedConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
