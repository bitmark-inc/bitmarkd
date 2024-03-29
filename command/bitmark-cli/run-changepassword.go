// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"github.com/urfave/cli"
)

func runChangePassword(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	// check existing password
	name, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if err != nil {
		return err
	}

	// prompt new password and confirm
	newPassword, err := promptNewPassword()
	if err != nil {
		return err
	}

	err = m.config.AddIdentity(name, owner.Description, owner.Seed, newPassword)
	if err != nil {
		return err
	}

	m.save = true
	return nil
}
