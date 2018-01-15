// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/urfave/cli"
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

	// optional existing hex key value
	privateKey, err := checkOptionalKey(c.String("privateKey"))
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "identity: %s\n", name)
		fmt.Fprintf(m.e, "description: %s\n", description)
	}

	err = addIdentity(m.config, name, description, privateKey, c.GlobalString("password"), m.testnet)
	if nil != err {
		return err
	}

	// require configuration update
	m.save = true
	return nil
}

func addIdentity(configs *configuration.Configuration, name string, description string, privateKeyStr string, password string, testnet bool) error {

	for _, identity := range configs.Identity {
		if name == identity.Name {
			return fmt.Errorf("identity: %q already exists", name)
		}
	}

	if "" == password {
		var err error
		// prompt password and pwd confirm for private key encryption
		password, err = promptPasswordReader()
		if nil != err {
			return err
		}
	}

	encrypted, privateKeyConfig, err := encrypt.MakeKeyPair(privateKeyStr, password, testnet)
	if nil != err {
		return err
	}

	identity := encrypt.IdentityType{
		Name:               name,
		Description:        description,
		Public_key:         encrypted.PublicKey,
		Seed:               encrypted.EncryptedSeed,
		Private_key:        encrypted.EncryptedPrivateKey,
		Private_key_config: *privateKeyConfig,
	}
	configs.Identity = append(configs.Identity, identity)

	return nil
}
