// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/urfave/cli"
)

func runProvenance(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	txId, err := checkTransferTxId(c.String("txid"))
	if nil != err {
		return err
	}

	count, err := checkRecordCount(c.String("count"))
	if nil != err {
		return err
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

	provenanceConfig := &rpccalls.ProvenanceData{
		TxId:  txId,
		Count: count,
	}

	response, err := client.GetProvenance(provenanceConfig)
	if nil != err {
		return err
	}

	printJson(m.w, response)

	return nil
}
