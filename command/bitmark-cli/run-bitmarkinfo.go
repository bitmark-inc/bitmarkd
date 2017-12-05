// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/urfave/cli"
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
