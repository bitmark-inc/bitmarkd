// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/urfave/cli"
)

func runInfo(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	infoConfig, err := configuration.GetInfoConfiguration(m.file, m.variables)
	if nil != err {
		return err
	}

	// add base58 Bitmark Account to output structure
	for i, id := range infoConfig.Identity {
		pub, err := hex.DecodeString(id.Public_key)
		if nil != err {
			return err
		}

		keyPair := &keypair.KeyPair{
			PublicKey: pub,
		}

		a := &account.Account{
			AccountInterface: &account.ED25519Account{
				Test:      m.testnet,
				PublicKey: keyPair.PublicKey[:],
			},
		}
		infoConfig.Identity[i].Account = a.String()

	}

	printJson(m.w, infoConfig)
	return nil
}
