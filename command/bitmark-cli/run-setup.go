// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
)

func runSetup(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)
	testnet := m.testnet

	name, err := checkName(c.GlobalString("identity"))
	if nil != err {
		return err
	}

	connect, err := checkConnect(c.String("connect"))
	if nil != err {
		return err
	}

	description, err := checkDescription(c.String("description"))
	if nil != err {
		return err
	}

	seed, err := checkSeed(c.String("seed"), c.Bool("new"), m.testnet)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "config: %s\n", m.file)
		fmt.Fprintf(m.e, "testnet: %t\n", testnet)
		fmt.Fprintf(m.e, "connect: %s\n", connect)
		fmt.Fprintf(m.e, "identity: %s\n", name)
		fmt.Fprintf(m.e, "description: %s\n", description)
	}

	// Create the folder hierarchy for configuration if not existing
	configDir := path.Dir(m.file)
	d, err := checkFileExists(configDir)
	if nil != err {
		if err = os.MkdirAll(configDir, 0750); nil != err {
			return err
		}
	} else if !d {
		return fmt.Errorf("path: %q is not a directory", configDir)
	}

	config := &configuration.Configuration{
		DefaultIdentity: name,
		TestNet:         testnet,
		Connections:     strings.Split(connect, ","),
		Identities:      make(map[string]configuration.Identity),
	}

	password := c.GlobalString("password")
	if "" == password {
		password, err = promptNewPassword()
		if nil != err {
			return err
		}
	}

	err = config.AddIdentity(name, description, seed, password)
	if nil != err {
		return err
	}

	m.config = config
	m.save = true

	return nil
}
