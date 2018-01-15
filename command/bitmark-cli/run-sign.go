// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"encoding/hex"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/keypair"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ed25519"
	"io/ioutil"
	"os"
)

func runSign(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	fileName, err := checkFileName(c.String("file"))
	if nil != err {
		return err
	}

	from, err := checkTransferFrom(c.GlobalString("identity"), m.config)
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "file: %s\n", fileName)
		fmt.Fprintf(m.e, "signer: %s\n", from.Name)
	}

	var ownerKeyPair *keypair.KeyPair

	// get global password items
	agent := c.GlobalString("agent")
	clearCache := c.GlobalBool("clearCache")
	password := c.GlobalString("password")

	// check owner password
	if "" != agent {
		password, err := passwordFromAgent(from.Name, "Transfer Bitmark", agent, clearCache)
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

	file, err := os.Open(fileName)
	if nil != err {
		return err
	}

	data, err := ioutil.ReadAll(file)

	signature := ed25519.Sign(ownerKeyPair.PrivateKey, data)
	s := hex.EncodeToString(signature)

	if m.verbose {
		fmt.Fprintf(m.e, "signature: %q\n", s)
	} else {

		out := struct {
			Identity  string `json:"identity"`
			FileName  string `json:"file_name"`
			Signature string `json:"signature"`
		}{
			Identity:  from.Name,
			FileName:  fileName,
			Signature: s,
		}
		printJson(m.w, out)
	}
	return nil
}
