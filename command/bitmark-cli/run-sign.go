// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/urfave/cli"
	"golang.org/x/crypto/ed25519"
)

func runSign(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	fileName, err := checkFileName(c.String("file"))
	if nil != err {
		return err
	}

	from, owner, err := checkOwnerWithPasswordPrompt(c.GlobalString("identity"), m.config, c)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "file: %s\n", fileName)
		fmt.Fprintf(m.e, "signer: %s\n", from)
	}

	file, err := os.Open(fileName)
	if nil != err {
		return err
	}

	data, err := ioutil.ReadAll(file)
	if nil != err {
		return err
	}

	signature := ed25519.Sign(owner.PrivateKey.PrivateKeyBytes(), data)
	s := hex.EncodeToString(signature)

	if m.verbose {
		fmt.Fprintf(m.e, "signature: %q\n", s)
	} else {

		out := struct {
			Identity  string `json:"identity"`
			FileName  string `json:"file_name"`
			Signature string `json:"signature"`
		}{
			Identity:  from,
			FileName:  fileName,
			Signature: s,
		}
		printJson(m.w, out)
	}
	return nil
}

func runVerify(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	fileName, err := checkFileName(c.String("file"))
	if nil != err {
		return err
	}

	from, owner, err := checkRecipient(c, "owner", m.config)
	if nil != err {
		return err
	}

	signature, err := checkSignature(c.String("signature"))
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "file: %s\n", fileName)
		fmt.Fprintf(m.e, "signer: %s\n", from)
		fmt.Fprintf(m.e, "signature: %x\n", signature)
	}

	file, err := os.Open(fileName)
	if nil != err {
		return err
	}

	data, err := ioutil.ReadAll(file)
	if nil != err {
		return err
	}

	ok := ed25519.Verify(owner.PublicKeyBytes(), data, signature)
	if m.verbose {
		fmt.Fprintf(m.e, "verified: %t\n", ok)
	} else {

		out := struct {
			Identity string `json:"identity"`
			FileName string `json:"file_name"`
			Verified bool   `json:"verified"`
		}{
			Identity: from,
			FileName: fileName,
			Verified: ok,
		}
		printJson(m.w, out)
	}
	return nil
}
