// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

func runKeyPair(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	identity, err := checkTransferFrom(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	var keyPair *keypair.KeyPair

	// check owner password
	if "" == c.GlobalString("password") {
		keyPair, err = promptAndCheckPassword(identity)
		if nil != err {
			return err
		}
	} else {
		keyPair, err = encrypt.VerifyPassword(c.GlobalString("password"), identity)
		if nil != err {
			return err
		}
	}
	//just in case some internal breakage
	if nil == keyPair {
		return fault.ErrKeyPairCannotBeNil
	}

	type KeyPairDisplay struct {
		Account    *account.Account    `json:"account"`
		PrivateKey *account.PrivateKey `json:"private_key"`
		KeyPair    keypair.RawKeyPair  `json:"raw"`
	}
	output := KeyPairDisplay{
		Account: &account.Account{
			AccountInterface: &account.ED25519Account{
				Test:      m.testnet,
				PublicKey: keyPair.PublicKey[:],
			},
		},

		PrivateKey: &account.PrivateKey{
			PrivateKeyInterface: &account.ED25519PrivateKey{
				Test:       m.testnet,
				PrivateKey: keyPair.PrivateKey[:],
			},
		},

		KeyPair: keypair.RawKeyPair{
			Seed:       keyPair.Seed,
			PublicKey:  hex.EncodeToString(keyPair.PublicKey[:]),
			PrivateKey: hex.EncodeToString(keyPair.PrivateKey[:]),
		},
	}
	printJson(m.w, output)
	return nil
}
