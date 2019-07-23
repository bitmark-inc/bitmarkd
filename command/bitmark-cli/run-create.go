// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
)

func runCreate(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	assetName := c.String("asset")

	fingerprint, err := checkAssetFingerprint(c.String("fingerprint"))
	if nil != err {
		return err
	}

	metadata, err := checkAssetMetadata(c.String("metadata"))
	if nil != err {
		return err
	}

	quantity := c.Int("quantity")
	if quantity <= 0 {
		return fmt.Errorf("invalid quantity: %d", quantity)
	}

	zeroNonceOnly := c.Bool("zero")
	if zeroNonceOnly && quantity != 1 {
		return fmt.Errorf("invalid free-issue quantity: %d only 1 is allowed", quantity)
	}

	name, registrant, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "issuer: %s\n", name)
		fmt.Fprintf(m.e, "assetName: %q\n", assetName)
		fmt.Fprintf(m.e, "fingerprint: %q\n", fingerprint)
		fmt.Fprintf(m.e, "metadata:\n")
		splitMeta := strings.Split(metadata, "\u0000")
		for i := 0; i < len(splitMeta); i += 2 {
			fmt.Fprintf(m.e, "  %q: %q\n", splitMeta[i], splitMeta[i+1])
		}
		fmt.Fprintf(m.e, "quantity: %d\n", quantity)
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connections[m.connectionOffset], m.verbose, m.e)
	if nil != err {
		return err
	}
	defer client.Close()

	assetConfig := &rpccalls.AssetData{
		Name:        assetName,
		Metadata:    metadata,
		Quantity:    quantity,
		Fingerprint: fingerprint,
		Registrant:  registrant,
	}

	assetResult, err := client.MakeAsset(assetConfig)
	if nil != err {
		return err
	}

	// make Issues
	issueConfig := &rpccalls.IssueData{
		Issuer:    assetConfig.Registrant,
		AssetId:   assetResult.AssetId,
		Quantity:  assetConfig.Quantity,
		FreeIssue: 1 == assetConfig.Quantity,
	}

	response, err := client.Issue(issueConfig)
	if issueConfig.FreeIssue && nil != err && strings.Contains(err.Error(), "transaction already exists") {
		// if free issue was already done, try again asking for payment
		if zeroNonceOnly {
			return err
		}
		issueConfig.FreeIssue = false
		response, err = client.Issue(issueConfig)
	}

	if nil != err {
		return err
	}

	printJson(m.w, response)
	return nil
}
