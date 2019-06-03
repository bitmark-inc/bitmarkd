// Copyright (c) 2014-2019 Bitmark Inc.
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

func runGrant(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	to, newOwnerKeyPair, err := checkTransferTo(c.String("receiver"), m.config)
	if nil != err {
		return err
	}

	from, err := checkTransferFrom(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	shareId, err := checkTxId(c.String("share-id"))
	if nil != err {
		return err
	}

	quantity := c.Uint64("quantity")
	if quantity == 0 {
		return fmt.Errorf("invalid quantity: %d", quantity)
	}

	beforeBlock := c.Uint64("before-block")

	if m.verbose {
		fmt.Fprintf(m.e, "shareId: %s\n", shareId)
		fmt.Fprintf(m.e, "quantity: %d\n", quantity)
		fmt.Fprintf(m.e, "sender: %s\n", from.Name)
		fmt.Fprintf(m.e, "receiver: %s\n", to)
		fmt.Fprintf(m.e, "beforeBlock: %d\n", beforeBlock)
	}

	var ownerKeyPair *keypair.KeyPair

	// get global password items
	agent := c.GlobalString("use-agent")
	clearCache := c.GlobalBool("zero-agent-cache")
	password := c.GlobalString("password")

	// check owner password
	if "" != agent {
		password, err := passwordFromAgent(from.Name, "Grant Shared Bitmark", agent, clearCache)
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

	grantConfig := &rpccalls.GrantData{
		ShareId:     shareId,
		Quantity:    quantity,
		Owner:       ownerKeyPair,
		Recipient:   newOwnerKeyPair,
		BeforeBlock: beforeBlock,
	}

	response, err := client.Grant(grantConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
