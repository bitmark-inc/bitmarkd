// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

func runProvenance(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	txId, err := checkTxId(c.String("txid"))
	if nil != err {
		return err
	}

	count := c.Int("count")
	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	if m.verbose {
		fmt.Fprintf(m.e, "txid: %s\n", txId)
		fmt.Fprintf(m.e, "count: %d\n", count)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connect, m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	// map for adding identity name to provenance records
	ids := make(map[string]string)
	for _, id := range m.config.Identities {
		pub, err := hex.DecodeString(id.PublicKey)
		if nil != err {
			return err
		}

		keyPair := &keypair.KeyPair{
			PublicKey: pub,
		}

		a := &account.Account{
			AccountInterface: &account.ED25519Account{
				Test:      m.testnet,
				PublicKey: keyPair.PublicKey[:],
			},
		}
		ids[a.String()] = id.Name
	}

	provenanceConfig := &rpccalls.ProvenanceData{
		TxId:       txId,
		Count:      count,
		Identities: ids,
	}

	response, err := client.GetProvenance(provenanceConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
