// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
	"golang.org/x/crypto/ed25519"
)

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version = "zero" // do not change this value

// tags for the signing key data
const (
	taggedSeed    = "SEED:"    // followed by base58 encoded seed as produced by desktop/cli client
	taggedPrivate = "PRIVATE:" // followed by 64 bytes of hex Ed25519 private key
)

// main program
func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	flags := []getoptions.Option{
		{Long: "help", HasArg: getoptions.NO_ARGUMENT, Short: 'h'},
		{Long: "verbose", HasArg: getoptions.NO_ARGUMENT, Short: 'v'},
		{Long: "quiet", HasArg: getoptions.NO_ARGUMENT, Short: 'q'},
		{Long: "livenet", HasArg: getoptions.NO_ARGUMENT, Short: 'l'},
		{Long: "testnet", HasArg: getoptions.NO_ARGUMENT, Short: 't'},
		{Long: "version", HasArg: getoptions.NO_ARGUMENT, Short: 'V'},
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if err != nil {
		exitwithstatus.Message("%s: getoptions error: %s", program, err)
	}

	if len(options["version"]) > 0 {
		exitwithstatus.Message("%s: version: %s", program, version)
	}

	if len(options["help"]) > 0 || len(arguments) == 0 {
		exitwithstatus.Message("usage: %s [--help] [--verbose] [--quiet] [--testnet|--livenet] files...]", program)
	}

	//verbose := len(options["verbose"]) > 0
	livenet := len(options["livenet"]) > 0
	testnet := len(options["testnet"]) > 0

	if testnet == livenet {
		exitwithstatus.Message("%s must select --livenet or --testnet", program)
	}

loop:
	for i, file := range arguments {

		n := i + 1
		fmt.Printf("%d: FILE:             %q\n", n, file)
		fmt.Printf("%d:\n", n)

		if databytes, err := ioutil.ReadFile(file); err != nil {
			fmt.Printf("%d: file: %q  error: %s\n", n, file, err)
		} else {
			rand := bytes.NewBuffer(databytes)
			publicKey, privateKey, err := ed25519.GenerateKey(rand)
			if err != nil {
				fmt.Printf("%d: public key generation  error: %s", n, err)
				continue loop
			}
			owner := &account.Account{
				AccountInterface: &account.ED25519Account{
					Test:      testnet,
					PublicKey: publicKey,
				},
			}
			fmt.Printf("%d: BADK owner:       %s\n", n, owner)
			fmt.Printf("%d: BADK public key:  %x\n", n, publicKey)
			fmt.Printf("%d: BADK private key: %x\n", n, privateKey)

			s := strings.TrimSpace(string(databytes))

			if strings.HasPrefix(s, taggedSeed) {
				privateKey, err := account.PrivateKeyFromBase58Seed(s[len(taggedSeed):])
				if err != nil {
					fmt.Printf("%d: private key generation  error: %s", n, err)
					continue loop
				}
				owner := privateKey.Account()
				publicKey := owner.PublicKeyBytes()

				fmt.Printf("%d: ----\n", n)
				fmt.Printf("%d: SEED owner:       %s\n", n, owner)
				fmt.Printf("%d: SEED public key:  %x\n", n, publicKey)
				fmt.Printf("%d: SEED private key: %x\n", n, privateKey.PrivateKeyBytes())

			} else if strings.HasPrefix(s, taggedPrivate) {
				b, err := hex.DecodeString(s[len(taggedPrivate):])
				if err != nil {
					fmt.Printf("%d: private key decode  error: %s", n, err)
					continue loop
				}
				privateKey, err := account.PrivateKeyFromBytes(b)
				if err != nil {
					fmt.Printf("%d: private key generation  error: %s", n, err)
					continue loop
				}
				owner := privateKey.Account()
				publicKey := owner.PublicKeyBytes()

				fmt.Printf("%d: ----\n", n)
				fmt.Printf("%d: PRIV owner:       %s\n", n, owner)
				fmt.Printf("%d: PRIV public key:  %x\n", n, publicKey)
				fmt.Printf("%d: PRIV private key: %x\n", n, privateKey.PrivateKeyBytes())

			}
			fmt.Printf("\n")
		}
	}
}
