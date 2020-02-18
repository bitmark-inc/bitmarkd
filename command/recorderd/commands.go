// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
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
//
// commands that run to create key and certificate files these
// commands cannot access any internal database or states or the
// configuration file
func processSetupCommand(program string, arguments []string) bool {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {
	case "generate-identity", "id":
		publicKeyFilename := getFilenameWithDirectory(arguments, recorderdPublicKeyFilename)
		privateKeyFilename := getFilenameWithDirectory(arguments, recorderdPrivateKeyFilename)

		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %v\n", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)

	case "start", "run":
		return false // continue processing

	case "version", "v":
		fmt.Printf("%s\n", version)

	default:
		switch command {
		case "help", "h", "?":
		case "", " ":
			fmt.Printf("error: missing command\n")
		default:
			fmt.Printf("error: no such command: %v\n", command)
		}

		fmt.Printf("usage: %s [--help] [--verbose] [--quiet] --config-file=FILE [[command|help] arguments...]", program)

		fmt.Printf("supported commands:\n\n")
		fmt.Printf("  help                       (h)      - display this message\n\n")
		fmt.Printf("  version                    (v)      - display version sting\n\n")

		fmt.Printf("  generate-identity [DIR]    (id)     - create private key in: %q\n", "DIR/"+recorderdPrivateKeyFilename)
		fmt.Printf("                                        and the public key in: %q\n", "DIR/"+recorderdPublicKeyFilename)
		fmt.Printf("\n")

		fmt.Printf("  start                      (run)    - just run the program, same as no arguments\n")
		fmt.Printf("                                        for convienience when passing script arguments\n")
		fmt.Printf("\n")

		exitwithstatus.Exit(1)
	}
	return true
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
