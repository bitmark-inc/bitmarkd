// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/urfave/cli"
)

func runAdd(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	name, err := checkName(c.GlobalString("identity"))
	if err != nil {
		return err
	}

	description, err := checkDescription(c.String("description"))
	if err != nil {
		return err
	}

	// blank or a valid seed
	seed := c.String("seed")
	newSeed := c.Bool("new")
	acc := c.String("account")

	if m.verbose {
		fmt.Fprintf(m.e, "identity: %s\n", name)
		fmt.Fprintf(m.e, "description: %s\n", description)
		fmt.Fprintf(m.e, "seed: %s\n", seed)
		fmt.Fprintf(m.e, "account: %s\n", acc)
		fmt.Fprintf(m.e, "new: %t\n", newSeed)
	}

	if acc == "" {
		seed, err = checkSeed(seed, newSeed, m.testnet)
		if err != nil {
			return err
		}

		password := c.GlobalString("password")
		if password == "" {
			password, err = promptNewPassword()
			if err != nil {
				return err
			}
		}

		err = m.config.AddIdentity(name, description, seed, password)
		if err != nil {
			return err
		}

	} else if seed == "" && acc != "" && !newSeed {
		err = m.config.AddReceiveOnlyIdentity(name, description, acc)
		if err != nil {
			return err
		}

	} else {
		return fault.IncompatibleOptions
	}

	// require configuration update
	m.save = true
	return nil
}
