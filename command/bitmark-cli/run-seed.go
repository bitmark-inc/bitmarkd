// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"strings"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
)

// merge the account string to private data
type seedResult struct {
	*configuration.Private
	Name    string `json:"name"`
	Account string `json:"account"`
	Phrase  string `json:"recovery_phrase"`
}

// to decrypt and show the secret data
func runSeed(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	name, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if nil != err {
		return err
	}

	phrase, err := account.Base58EncodedSeedToPhrase(owner.Seed)
	if nil != err {
		return err
	}

	result := seedResult{
		Private: owner,
		Name:    name,
		Account: owner.PrivateKey.Account().String(),
		Phrase:  strings.Join(phrase, " "),
	}

	printJson(m.w, result)
	return nil
}
