// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/urfave/cli"
)

func runGenerate(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	rawKeyPair, _, err := keypair.MakeRawKeyPair(m.testnet)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "rawKeyPair: %#v\n", rawKeyPair)
	}

	printJson(m.w, rawKeyPair)
	return nil
}
