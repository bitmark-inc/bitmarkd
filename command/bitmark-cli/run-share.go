// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

func runShare(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	from, err := checkIdentity(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	txId, err := checkTxId(c.String("txid"))
	if nil != err {
		return err
	}

	quantity := c.Int("quantity")
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity: %d", quantity)
	}

	if m.verbose {
		fmt.Fprintf(m.e, "txid: %s\n", txId)
		fmt.Fprintf(m.e, "quantity: %d\n", quantity)
	}

	var ownerKeyPair *keypair.KeyPair

	// get global password items
	agent := c.GlobalString("use-agent")
	clearCache := c.GlobalBool("zero-agent-cache")
	password := c.GlobalString("password")

	// check owner password
	if "" != agent {
		password, err := passwordFromAgent(from.Name, "Create Shareal Bitmark", agent, clearCache)
		if nil != err {
			return err
		}
		ownerKeyPair, err = encrypt.VerifyPassword(password, from)
		if nil != err {
			return err
		}
	} else if "" != password {
		ownerKeyPair, err = encrypt.VerifyPassword(password, from)
		if nil != err {
			return err
		}
	} else {
		ownerKeyPair, err = promptAndCheckPassword(from)
		if nil != err {
			return err
		}

	}
	// just in case some internal breakage
	if nil == ownerKeyPair {
		return ErrNilKeyPair
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connect, m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	// make Share
	shareConfig := &rpccalls.ShareData{
		Owner:    ownerKeyPair,
		TxId:     txId,
		Quantity: uint64(quantity),
	}

	response, err := client.Share(shareConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)
	return nil
}
