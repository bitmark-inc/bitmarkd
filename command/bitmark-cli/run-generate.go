// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/keypair"
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
