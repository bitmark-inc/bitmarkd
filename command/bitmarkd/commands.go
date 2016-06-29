// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	//"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
	"os"
	"strconv"
)

// setup command handler
// commands that run to create key and certificate files
// these commands cannot access any internal database or states
func processSetupCommand(log *logger.L, arguments []string, options *Configuration) bool {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {
	case "generate-peer-identity", "peer":
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

	case "generate-rpc-cert", "rpc":
		certificateFilename := options.ClientRPC.Certificate
		privateKeyFilename := options.ClientRPC.PrivateKey
		addresses := []string{}
		if len(arguments) >= 2 {
			for _, a := range arguments[1:] {
				if "" != a {
					addresses = append(addresses, a)
				}
			}
		}
		if len(arguments) >= 1 && "" != arguments[0] {
			certificateFilename = arguments[0] + ".crt"
			privateKeyFilename = arguments[0] + ".key"
		}
		err := makeSelfSignedCertificate("rpc", certificateFilename, privateKeyFilename, 0 != len(addresses), addresses)
		if nil != err {
			fmt.Printf("cannot generate RPC key: %q and certificate: %q\n", privateKeyFilename, certificateFilename)
			log.Criticalf("cannot generate RPC key: %q and certificate: %q", privateKeyFilename, certificateFilename)
			fmt.Printf("error generating RPC key/certificate: %v\n", err)
			log.Criticalf("error generating RPC key/certificate: %v", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated RPC key: %q and certificate: %q\n", privateKeyFilename, certificateFilename)
		log.Infof("generated RPC key: %q and certificate: %q", privateKeyFilename, certificateFilename)

	case "generate-proof-identity", "proof":
		publicKeyFilename := options.Proofing.PublicKey
		privateKeyFilename := options.Proofing.PrivateKey

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

	case "block-times":
		return false // defer processing until database is loaded

	default:
		switch command {
		case "help", "h", "?":
		case "", " ":
			fmt.Printf("error: missing command\n")
		default:
			fmt.Printf("error: no such command: %v\n", command)
		}

		fmt.Printf("supported commands:\n\n")
		fmt.Printf("  help                   (h)       - display this message\n\n")

		fmt.Printf("  gen-peer-identity      (peer)    - create private key in: %q\n", options.Peering.PrivateKey)
		fmt.Printf("                                     and the public key in: %q\n", options.Peering.PublicKey)
		fmt.Printf("\n")

		fmt.Printf("  gen-rpc-cert           (rpc)     - create private key in:  %q\n", options.ClientRPC.PrivateKey)
		fmt.Printf("                                     and the certificate in: %q\n", options.ClientRPC.Certificate)
		fmt.Printf("\n")

		fmt.Printf("  gen-rpc-cert PREFIX IPs...       - create private key in: '<PREFIX>.key'\n")
		fmt.Printf("                                     and the certificate in '<PREFIX>.crt'\n")
		fmt.Printf("\n")

		fmt.Printf("  gen-proof-identity     (proof)   - create private key in:  %q\n", options.Proofing.PrivateKey)
		fmt.Printf("                                     and the certificate in: %q\n", options.Proofing.PublicKey)
		fmt.Printf("\n")

		fmt.Printf("  block-times FILE BEGIN END       - write time and difficulty to text file for a range of blocks\n")
		exitwithstatus.Exit(1)
	}

	// indicate processing complete and prefor normal exit from main
	return true
}

// data command handler
// the internal block and transaction pools are enabled so these commands can
// access and/or change these databases
func processDataCommand(log *logger.L, arguments []string, options *Configuration) bool {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {

	case "block-times":
		if len(arguments) < 3 {
			fmt.Printf("missing arguments arguments (use '' for stdout, and '0' for min/max)\n")
			exitwithstatus.Exit(1)
		}

		begin, err := strconv.ParseUint(arguments[1], 10, 64)
		if nil != err {
			fmt.Printf("error in begin block number: %v\n", err)
			exitwithstatus.Exit(1)
		}
		end, err := strconv.ParseUint(arguments[2], 10, 64)
		if nil != err {
			fmt.Printf("error in end block number: %v\n", err)
			exitwithstatus.Exit(1)
		}

		fmt.Printf("*********** ERROR: %d %d\n", begin, end) // ***** FIX THIS: remove later

		switch filename := arguments[0]; filename {
		case "": // use stdout
			fallthrough
		case "-": // use stdout
			// block.PrintBlockTimes(os.Stdout, begin, end)
			panic("HERE")

		default:
			fh, err := os.Create(filename)

			if nil != err {
				fmt.Printf("cannot create: %q  error: %v\n", filename, err)
				exitwithstatus.Exit(1)
			}
			defer fh.Close()
			//block.PrintBlockTimes(fh, begin, end)
			panic("HERE")
		}

	default:
		fmt.Printf("error: no such command: %v\n", command)
		exitwithstatus.Exit(1)

	}

	// indicate processing complete and prefor normal exit from main
	return true
}
