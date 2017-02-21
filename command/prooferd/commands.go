// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
)

// setup command handler
// commands that run to create key and certificate files
// these commands cannot access any internal database or states
func processSetupCommand(log *logger.L, arguments []string, options *Configuration) {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {
	case "generate-identity":
		publicKeyFilename := options.Peering.PublicKey
		privateKeyFilename := options.Peering.PrivateKey

		if len(arguments) >= 1 && "" != arguments[0] {
			publicKeyFilename = arguments[0] + ".public"
			privateKeyFilename = arguments[0] + ".private"
		}
		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			log.Criticalf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %v\n", err)
			log.Criticalf("error generating server key pair: %v\n", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
		log.Infof("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)

	default:
		switch command {
		case "help", "h", "?":
		case "", " ":
			fmt.Printf("error: missing command\n")
		default:
			fmt.Printf("error: no such command: %v\n", command)
		}

		fmt.Printf("supported commands:\n\n")
		fmt.Printf("  help                             - display this message\n\n")

		fmt.Printf("  generate-identity                - create private key in: %q\n", options.Peering.PrivateKey)
		fmt.Printf("                                     and the public key in: %q\n", options.Peering.PublicKey)
		fmt.Printf("\n")

		exitwithstatus.Exit(1)
	}
}
