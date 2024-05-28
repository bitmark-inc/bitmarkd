// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/base64"
	"fmt"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/urfave/cli"
)

func runBlockDecode(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	packed := c.String("packed")
	if packed == "" {
		return fmt.Errorf("empty packed")
	}

	p64, err := base64.StdEncoding.DecodeString(packed)
	if err != nil {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "packed: %s\n", packed)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if err != nil {
		return err
	}
	defer client.Close()

	blockDecodeConfig := &rpccalls.BlockDecodeData{
		Packed: p64,
	}

	response, err := client.DecodeBlock(blockDecodeConfig)
	if err != nil {
		return err
	}

	printJson(m.w, response)

	return nil
}
