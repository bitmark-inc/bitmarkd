// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
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
	if nil != err {
		return err
	}

	// prompt new password and confirm
	newPassword, err := promptNewPassword()
	if nil != err {
		return err
	}

	err = m.config.AddIdentity(name, owner.Description, owner.Seed, newPassword)
	if nil != err {
		return err
	}

	m.save = true
	return nil
}
