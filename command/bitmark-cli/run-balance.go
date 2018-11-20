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

func runBalance(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	shareId := c.String("share-id")

	owner := c.String("owner")
	if "" == owner {
		owner = c.GlobalString("identity")
		if "" == owner {
			owner = m.config.DefaultIdentity
		}
	}

	count := c.Int("count")
	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	if m.verbose {
		fmt.Fprintf(m.e, "owner: %s\n", owner)
		fmt.Fprintf(m.e, "shareId: %s\n", shareId)
		fmt.Fprintf(m.e, "count: %d\n", count)
	}

	var ownerKeyPair *keypair.KeyPair

	// ***** FIX THIS: possibly add base58 keys @@@@@
	publicKey, err := hex.DecodeString(owner)
	if nil != err {

		ownerKeyPair, err = encrypt.PublicKeyFromIdentity(owner, m.config.Identities)
		if nil != err {
			return err
		}
	} else {
		ownerKeyPair = &keypair.KeyPair{}
		if len(publicKey) != encrypt.PublicKeySize {
			return err
		}
		ownerKeyPair.PublicKey = publicKey
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

	balanceConfig := &rpccalls.BalanceData{
		Owner:   ownerKeyPair,
		ShareId: shareId,
		Count:   count,
	}

	response, err := client.GetBalance(balanceConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
