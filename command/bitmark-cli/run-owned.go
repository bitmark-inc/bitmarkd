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

func runOwned(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	ownerId := c.String("owner")
	if "" == ownerId {
		ownerId = c.GlobalString("identity")
		if "" == ownerId {
			ownerId = m.config.DefaultIdentity
		}
	}

	start := c.Uint64("start")

	count := c.Int("count")
	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	if m.verbose {
		fmt.Fprintf(m.e, "owner: %s\n", ownerId)
		fmt.Fprintf(m.e, "start: %d\n", start)
		fmt.Fprintf(m.e, "count: %d\n", count)
	}

	owner, err := m.config.Account(ownerId)
	if nil != err {
		return err
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[0], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	ownedConfig := &rpccalls.OwnedData{
		Owner: owner,
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
