// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/urfave/cli"
)

func runCountersign(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	hex, err := checkTransferTx(c.String("transfer"))
	if nil != err {
		return err
	}

	// this command is run by the receiver so from is used
	// to get default identity
	to, err := checkTransferFrom(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "tx: %s\n", hex)
		fmt.Fprintf(m.e, "receiver: %s\n", to)
	}

	var newOwnerKeyPair *keypair.KeyPair

	// get global password items
	agent := c.GlobalString("agent")
	clearCache := c.GlobalBool("clearCache")
	password := c.GlobalString("password")

	// check owner password
	if "" != agent {
		password, err := passwordFromAgent(to.Name, "Transfer Bitmark", agent, clearCache)
		if nil != err {
			return err
		}
		newOwnerKeyPair, err = encrypt.VerifyPassword(password, to)
		if nil != err {
			return err
		}
	} else if "" != password {
		newOwnerKeyPair, err = encrypt.VerifyPassword(password, to)
		if nil != err {
			return err
		}
	} else {
		newOwnerKeyPair, err = promptAndCheckPassword(to)
		if nil != err {
			return err
		}

	}
	// just in case some internal breakage
	if nil == newOwnerKeyPair {
		return ErrNilKeyPair
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connect, m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	countersignConfig := &rpccalls.CountersignData{
		Transfer: hex,
		NewOwner: newOwnerKeyPair,
	}

	response, err := client.CountersignTransfer(countersignConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
