// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/urfave/cli"
)

func runAccount(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	publicKey, err := checkPublicKey(c.String("publickey"))
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "publicKey: %s\n", publicKey)
	}

	account, err := keypair.AccountFromHexPublicKey(publicKey, m.testnet)
	if nil != err {
		return err
	}

	result := struct {
		Hex    string `json:"hex"`
		Base58 string `json:"account"`
	}{
		Hex:    publicKey,
		Base58: account.String(),
	}

	printJson(m.w, result)
	return nil
}
