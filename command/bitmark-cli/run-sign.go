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

	"github.com/bitmark-inc/bitmarkd/command/bitmark-cli/encrypt"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/keypair"
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
	agent := c.GlobalString("use-agent")
	clearCache := c.GlobalBool("zero-agent-cache")
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
		return fault.ErrKeyPairCannotBeNil
	}

	file, err := os.Open(fileName)
	if nil != err {
		return err
	}

	data, err := ioutil.ReadAll(file)
	if nil != err {
		return err
	}

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

func runVerify(c *cli.Context) error {

	m := c.App.Metadata["config"].(*metadata)

	fileName, err := checkFileName(c.String("file"))
	if nil != err {
		return err
	}

	owner, ownerKeyPair, err := checkTransferTo(c.String("owner"), m.config)
	if nil != err {
		return err
	}

	signature, err := checkSignature(c.String("signature"))
	if nil != err {
		return err
	}

	if m.verbose {
		fmt.Fprintf(m.e, "file: %s\n", fileName)
		fmt.Fprintf(m.e, "signer: %s\n", owner)
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

	ok := ed25519.Verify(ownerKeyPair.PublicKey, data, signature)
	if m.verbose {
		fmt.Fprintf(m.e, "verified: %t\n", ok)
	} else {

		out := struct {
			Identity string `json:"identity"`
			FileName string `json:"file_name"`
			Verified bool   `json:"verified"`
		}{
			Identity: owner,
			FileName: fileName,
			Verified: ok,
		}
		printJson(m.w, out)
	}
	return nil
}
