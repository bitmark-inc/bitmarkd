// Copyright (c) 2014-2019 Bitmark Inc.
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

func runOwned(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	owner := c.String("owner")
	if "" == owner {
		owner = c.GlobalString("identity")
		if "" == owner {
			owner = m.config.DefaultIdentity
		}
	}

	start := c.Uint64("start")
	if start < 0 {
		return fmt.Errorf("invalid start: %d", start)
	}

	count := c.Int("count")
	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	if m.verbose {
		fmt.Fprintf(m.e, "owner: %s\n", owner)
		fmt.Fprintf(m.e, "start: %d\n", start)
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

	ownedConfig := &rpccalls.OwnedData{
		Owner: ownerKeyPair,
		Start: start,
		Count: count,
	}

	response, err := client.GetOwned(ownedConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
