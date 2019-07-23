// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
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

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	// map for adding identity name to provenance records
	ids := make(map[string]string)
	for name, id := range m.config.Identities {
		ids[id.Account] = name
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
