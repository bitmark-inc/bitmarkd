// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

func runSwap(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	to, err := checkTransferTo(c.String("receiver"))
	if nil != err {
		return err
	}

	from, err := checkTransferFrom(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	shareIdOne, err := checkTxId(c.String("share-id-one"))
	if nil != err {
		return err
	}

	quantityOne := c.Uint64("quantity-one")
	if quantityOne == 0 {
		return fmt.Errorf("invalid quantity-one: %d", quantityOne)
	}

	shareIdTwo, err := checkTxId(c.String("share-id-two"))
	if nil != err {
		return err
	}

	quantityTwo := c.Uint64("quantity-two")
	if quantityTwo == 0 {
		return fmt.Errorf("invalid quantity-two: %d", quantityTwo)
	}

	beforeBlock := c.Uint64("before-block")

	if m.verbose {
		fmt.Fprintf(m.e, "shareIdOne: %s\n", shareIdOne)
		fmt.Fprintf(m.e, "quantityOne: %d\n", quantityOne)
		fmt.Fprintf(m.e, "ownerOne: %s\n", from.Name)
		fmt.Fprintf(m.e, "shareIdTwo: %s\n", shareIdTwo)
		fmt.Fprintf(m.e, "quantityTwo: %d\n", quantityTwo)
		fmt.Fprintf(m.e, "ownerTwo: %s\n", to)
		fmt.Fprintf(m.e, "beforeBlock: %d\n", beforeBlock)
	}

	var ownerKeyPair *keypair.KeyPair

	// get global password items
	agent := c.GlobalString("use-agent")
	clearCache := c.GlobalBool("zero-agent-cache")
	password := c.GlobalString("password")

	// check owner password
	if "" != agent {
		password, err := passwordFromAgent(from.Name, "Swap Shareal Bitmark", agent, clearCache)
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

	var newOwnerKeyPair *keypair.KeyPair

	// ***** FIX THIS: possibly add base58 keys @@@@@
	newPublicKey, err := hex.DecodeString(to)
	if nil != err {

		newOwnerKeyPair, err = encrypt.PublicKeyFromIdentity(to, m.config.Identities)
		if nil != err {
			return err
		}
	} else {
		newOwnerKeyPair = &keypair.KeyPair{}
		if len(newPublicKey) != encrypt.PublicKeySize {
			return err
		}
		newOwnerKeyPair.PublicKey = newPublicKey
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

	swapConfig := &rpccalls.SwapData{
		ShareIdOne:  shareIdOne,
		QuantityOne: quantityOne,
		OwnerOne:    ownerKeyPair,
		ShareIdTwo:  shareIdTwo,
		QuantityTwo: quantityTwo,
		OwnerTwo:    newOwnerKeyPair,
		BeforeBlock: beforeBlock,
	}

	response, err := client.Swap(swapConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
