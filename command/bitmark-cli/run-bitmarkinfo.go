// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
)

func runBitmarkInfo(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	client, err := rpccalls.NewClient(m.testnet, m.config.Connect, m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	response, err := client.GetBitmarkInfo()
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
