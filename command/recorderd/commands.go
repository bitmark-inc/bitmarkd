// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"path/filepath"

	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
)

const (
	recorderdPublicKeyFilename  = "recorderd.public"
	recorderdPrivateKeyFilename = "recorderd.private"
)

// setup command handler
// commands that run to create key and certificate files
// these commands cannot access any internal database or states
func processSetupCommand(arguments []string) {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {
	case "generate-identity":
		publicKeyFilename := getFilenameWithDirectory(arguments, recorderdPublicKeyFilename)
		privateKeyFilename := getFilenameWithDirectory(arguments, recorderdPrivateKeyFilename)

		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %v\n", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)

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

		fmt.Printf("  generate-identity [DIR]          - create private key in: %q\n", "DIR/"+recorderdPrivateKeyFilename)
		fmt.Printf("                                     and the public key in: %q\n", "DIR/"+recorderdPublicKeyFilename)
		fmt.Printf("\n")

		exitwithstatus.Exit(1)
	}
}

// get the working directory; if not set in the arguments
// it's set to the current directory
func getFilenameWithDirectory(arguments []string, name string) string {
	dir := "."
	if len(arguments) >= 1 {
		dir = arguments[0]
	}

	return filepath.Join(dir, name)
}
