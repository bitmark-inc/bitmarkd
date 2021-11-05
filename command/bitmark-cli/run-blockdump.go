// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/urfave/cli"
)

func runBlockDump(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	block := c.Uint64("start")
	if block < 1 {
		return fmt.Errorf("invalid start block: %d", block)
	}

	count := c.Int("count")
	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	txs := c.Bool("txs")
	if m.verbose {
		fmt.Fprintf(m.e, "start block: %d\n", block)
		fmt.Fprintf(m.e, "count: %d\n", count)
		fmt.Fprintf(m.e, "decode txs: %t\n", txs)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	blockDumpConfig := &rpccalls.BlockDumpData{
		Block: block,
		Count: count,
		Txs:   txs,
	}

	response, err := client.GetBlocks(blockDumpConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
