// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/urfave/cli"
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
		return ErrNilKeyPair
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
	if encrypted.PublicKey != identity.Public_key {
		return err
	}
	identity.Seed = encrypted.EncryptedSeed
	identity.Private_key = encrypted.EncryptedPrivateKey
	identity.Private_key_config = *privateKeyConfig

	m.save = true
	return nil
}
