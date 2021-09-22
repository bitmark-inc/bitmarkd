// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2021 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/urfave/cli"
)

func runFullProvenance(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	bitmarkId, err := checkTxId(c.String("bitmarkid"))
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "bitmark id: %s\n", bitmarkId)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	// map for adding identity name to full provenance records
	ids := make(map[string]string)
	for name, id := range m.config.Identities {
		ids[id.Account] = name
	}

	fullProvenanceConfig := &rpccalls.FullProvenanceData{
		BitmarkId:  bitmarkId,
		Identities: ids,
	}

	response, err := client.GetFullProvenance(fullProvenanceConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
