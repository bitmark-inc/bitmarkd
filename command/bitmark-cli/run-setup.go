// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/configuration"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/urfave/cli"
	"os"
	"strings"
)

func runSetup(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	name, err := checkName(c.GlobalString("identity"))
	if nil != err {
		return err
	}

	network, err := checkNetwork(c.String("network"))
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

	// optional existing hex key value
	privateKey, err := checkOptionalKey(c.String("privateKey"))
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "config: %s\n", m.file)
		fmt.Fprintf(m.e, "network: %s\n", network)
		fmt.Fprintf(m.e, "connect: %s\n", connect)
		fmt.Fprintf(m.e, "identity: %s\n", name)
		fmt.Fprintf(m.e, "description: %s\n", description)
	}

	// Create the folder hierarchy for configuration if not existing
	folderIndex := strings.LastIndex(m.file, "/")
	if folderIndex >= 0 {
		configDir := m.file[:folderIndex]
		if !ensureFileExists(configDir) {
			if err := os.MkdirAll(configDir, 0755); nil != err {
				return err
			}
		}
	}
	configData := &configuration.Configuration{
		Default_identity: name,
		Network:          network,
		Connect:          connect,
		Identity:         make([]encrypt.IdentityType, 0),
	}

	// flag to indicate testnet keys
	testnet := "bitmark" != configData.Network

	err = addIdentity(configData, name, description, privateKey, c.GlobalString("password"), testnet)
	if nil != err {
		return err
	}

	m.config = configData
	m.save = true

	return nil
}
