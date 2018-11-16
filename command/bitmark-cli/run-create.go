// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/rpccalls"
	"github.com/bitmark-inc/bitmarkd/keypair"
)

func runCreate(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	issuer, err := checkIdentity(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	assetName, err := checkAssetName(c.String("asset"))
	if nil != err {
		return err
	}

	fingerprint, err := checkAssetFingerprint(c.String("fingerprint"))
	if nil != err {
		return err
	}

	metadata, err := checkAssetMetadata(c.String("metadata"))
	if nil != err {
		return err
	}

	quantity, err := checkAssetQuantity(c.String("quantity"))
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "issuer: %s\n", issuer.Name)
		fmt.Fprintf(m.e, "assetName: %q\n", assetName)
		fmt.Fprintf(m.e, "fingerprint: %q\n", fingerprint)
		fmt.Fprintf(m.e, "metadata:\n")
		splitMeta := strings.Split(metadata, "\u0000")
		for i := 0; i < len(splitMeta); i += 2 {
			fmt.Fprintf(m.e, "  %q: %q\n", splitMeta[i], splitMeta[i+1])
		}
		fmt.Fprintf(m.e, "quantity: %d\n", quantity)
	}

	var registrant *keypair.KeyPair

	// get global password items
	agent := c.GlobalString("use-agent")
	clearCache := c.GlobalBool("zero-agent-cache")
	password := c.GlobalString("password")

	// check password
	if "" != agent {
		password, err := passwordFromAgent(issuer.Name, "Create Bitmark", agent, clearCache)
		if nil != err {
			return err
		}
		registrant, err = encrypt.VerifyPassword(password, issuer)
		if nil != err {
			return err
		}
	} else if "" != password {
		registrant, err = encrypt.VerifyPassword(password, issuer)
		if nil != err {
			return err
		}
	} else {
		registrant, err = promptAndCheckPassword(issuer)
		if nil != err {
			return err
		}
	}
	// just in case some internal breakage
	if nil == registrant {
		return ErrNilKeyPair
	}

	client, err := rpccalls.NewClient(m.testnet, m.config.Connect, m.verbose, m.e)
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
		issueConfig.FreeIssue = false
		response, err = client.Issue(issueConfig)
	}

	if nil != err {
		return err
	}

	printJson(m.w, response)
	return nil
}
