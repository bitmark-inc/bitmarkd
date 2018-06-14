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
	"github.com/bitmark-inc/bitmarkd/currency"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

func runBlockTransfer(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	txId, err := checkTransferTxId(c.String("txid"))
	if nil != err {
		return err
	}

	to, err := checkTransferTo(c.String("receiver"))
	if nil != err {
		return err
	}

	bitcoinAddress, err := checkCoinAddress(currency.Bitcoin, c.String("bitcoin"), m.testnet)
	if nil != err {
		return err
	}
	litecoinAddress, err := checkCoinAddress(currency.Litecoin, c.String("litecoin"), m.testnet)
	if nil != err {
		return err
	}

	from, err := checkTransferFrom(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	payments := currency.Map{
		currency.Bitcoin:  bitcoinAddress,
		currency.Litecoin: litecoinAddress,
	}

	if m.verbose {
		fmt.Fprintf(m.e, "txid: %s\n", txId)
		fmt.Fprintf(m.e, "receiver: %s\n", to)
		fmt.Fprintf(m.e, "sender: %s\n", from.Name)
	}

	var ownerKeyPair *keypair.KeyPair

	// get global password items
	agent := c.GlobalString("use-agent")
	clearCache := c.GlobalBool("zero-agent-cache")
	password := c.GlobalString("password")

	// check owner password
	if "" != agent {
		password, err := passwordFromAgent(from.Name, "Transfer Block Owner", agent, clearCache)
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

		newOwnerKeyPair, err = encrypt.PublicKeyFromIdentity(to, m.config.Identity)
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

	transferConfig := &rpccalls.BlockTransferData{
		Owner:    ownerKeyPair,
		NewOwner: newOwnerKeyPair,
		Payments: payments,
		TxId:     txId,
	}

	response, err := client.SingleSignedBlockTransfer(transferConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
