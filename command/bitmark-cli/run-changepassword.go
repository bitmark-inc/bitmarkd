// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

func runChangePassword(c *cli.Context) error {

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

	// prompt new password and pwd confirm for private key encryption
	newPassword, err := promptPasswordReader()
	if nil != err {
		return err
	}

	input := ""
	if 0 == len(keyPair.Seed) {
		input = hex.EncodeToString(keyPair.PrivateKey[:])
	} else {
		input = "SEED:" + keyPair.Seed
	}

	encrypted, privateKeyConfig, err := encrypt.MakeKeyPair(input, newPassword, m.testnet)
	if nil != err {
		return err
	}
	if encrypted.PublicKey != identity.PublicKey {
		return err
	}
	identity.Seed = encrypted.EncryptedSeed
	identity.PrivateKey = encrypted.EncryptedPrivateKey
	identity.PrivateKeyConfig = *privateKeyConfig

	m.save = true
	return nil
}
