// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/fault"
)

func runAdd(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	name, err := checkName(c.GlobalString("identity"))
	if nil != err {
		return err
	}

	description, err := checkDescription(c.String("description"))
	if nil != err {
		return err
	}

	// blank or a valid seed
	seed := c.String("seed")
	new := c.Bool("new")
	acc := c.String("account")

	if m.verbose {
		fmt.Fprintf(m.e, "identity: %s\n", name)
		fmt.Fprintf(m.e, "description: %s\n", description)
		fmt.Fprintf(m.e, "seed: %s\n", seed)
		fmt.Fprintf(m.e, "account: %s\n", acc)
		fmt.Fprintf(m.e, "new: %t\n", new)
	}

	if "" == acc {
		seed, err = checkSeed(seed, new, m.testnet)
		if nil != err {
			return err
		}

		password := c.GlobalString("password")
		if "" == password {
			password, err = promptNewPassword()
			if nil != err {
				return err
			}
		}

		err = m.config.AddIdentity(name, description, seed, password)
		if nil != err {
			return err
		}

	} else if "" == seed && "" != acc && !new {
		err = m.config.AddReceiveOnlyIdentity(name, description, acc)
		if nil != err {
			return err
		}

	} else {
		return fault.IncompatibleOptions
	}

	// require configuration update
	m.save = true
	return nil
}
