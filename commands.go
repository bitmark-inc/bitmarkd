// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
)

// command handler
func processCommand(log *logger.L, options configuration.CommandOptions) {

	command := options.Args.Command
	arguments := options.Args.Arguments

	switch command {
	case "generate-identity":
		publicKeyFilename := options.PublicKey
		privateKeyFilename := options.PrivateKey

		if len(arguments) >= 1 && "" != arguments[0] {
			publicKeyFilename = arguments[0] + ".public"
			privateKeyFilename = arguments[0] + ".private"
		}
		err := makeKeyPair("rpc", publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: '%s' and public key: '%s'\n", privateKeyFilename, publicKeyFilename)
			log.Criticalf("cannot generate private key: '%s' and public key: '%s'\n", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %v\n", err)
			log.Criticalf("error generating server key pair: %v\n", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated private key: '%s' and public key: '%s'\n", privateKeyFilename, publicKeyFilename)
		log.Infof("generated private key: '%s' and public key: '%s'\n", privateKeyFilename, publicKeyFilename)

	case "generate-rpc-cert":
		certificateFilename := options.RPCCertificate
		privateKeyFilename := options.RPCKey
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
			fmt.Printf("cannot generate RPC key: '%s' and certificate: '%s'\n", privateKeyFilename, certificateFilename)
			log.Criticalf("cannot generate RPC key: '%s' and certificate: '%s'", privateKeyFilename, certificateFilename)
			fmt.Printf("error generating RPC key/certificate: %v\n", err)
			log.Criticalf("error generating RPC key/certificate: %v", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated RPC key: '%s' and certificate: '%s'\n", privateKeyFilename, certificateFilename)
		log.Infof("generated RPC key: '%s' and certificate: '%s'", privateKeyFilename, certificateFilename)

	// case "generate-peer-cert" == command:
	// 	certificateFilename := options.PeerCertificate
	// 	privateKeyFilename := options.PeerKey
	// 	addresses := []string{}
	// 	if len(arguments) >= 2 {
	// 		for _, a := range arguments[1:] {
	// 			if "" != a {
	// 				addresses = append(addresses, a)
	// 			}
	// 		}
	// 	}
	// 	if len(arguments) >= 1 && "" != arguments[0] {
	// 		certificateFilename = arguments[0] + ".crt"
	// 		privateKeyFilename = arguments[0] + ".key"
	// 	}
	// 	err := makeSelfSignedCertificate("peer", certificateFilename, privateKeyFilename, 0 != len(addresses), addresses)
	// 	if nil != err {
	// 		fmt.Printf("cannot generate peer key: '%s' and certificate: '%s'\n", privateKeyFilename, certificateFilename)
	// 		log.Criticalf("cannot generate peer key: '%s' and certificate: '%s'", privateKeyFilename, certificateFilename)
	// 		fmt.Printf("error generating peer key/certificate: %v\n", err)
	// 		log.Criticalf("error generating peer key/certificate: %v", err)
	// 		exitwithstatus.Exit(1)
	// 	}
	// 	fmt.Printf("generated peer key: '%s' and certificate: '%s'\n", privateKeyFilename, certificateFilename)
	// 	log.Infof("generated peer key: '%s' and certificate: '%s'", privateKeyFilename, certificateFilename)

	case "generate-mine-cert":
		certificateFilename := options.MineCertificate
		privateKeyFilename := options.MineKey
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
		err := makeSelfSignedCertificate("mine", certificateFilename, privateKeyFilename, 0 != len(addresses), addresses)
		if nil != err {
			fmt.Printf("cannot generate mine key: '%s' and certificate: '%s'\n", privateKeyFilename, certificateFilename)
			log.Criticalf("cannot generate mine key: '%s' and certificate: '%s'", privateKeyFilename, certificateFilename)
			fmt.Printf("error generating mine key/certificate: %v\n", err)
			log.Criticalf("error generating mine key/certificate: %v", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated mine key: '%s' and certificate: '%s'\n", privateKeyFilename, certificateFilename)
		log.Infof("generated mine key: '%s' and certificate: '%s'", privateKeyFilename, certificateFilename)

	default:
		if "help" != command {
			fmt.Printf("error: no such command: %v\n", command)
		}
		fmt.Printf("commands:\n")
		fmt.Printf("  generate-identity                - create server private key in: '%s' and public key in: '%s'\n", options.PrivateKey, options.PublicKey)
		fmt.Printf("  generate-rpc-cert                - create private key in: '%s' and certificate in: '%s'\n", options.RPCKey, options.RPCCertificate)
		fmt.Printf("  generate-rpc-cert PREFIX IPs...  - create private key in: '<PREFIX>.key' certificate in '<PREFIX>.crt'\n")
		//fmt.Printf("  generate-peer-cert               - create private key in: '%s' and certificate in: '%s'\n", options.PeerKey, options.PeerCertificate)
		//fmt.Printf("  generate-peer-cert PREFIX IPs... - create private key in: '<PREFIX>.key' certificate in: '<PREFIX>.crt'\n")
		fmt.Printf("  generate-mine-cert               - create private key in: '%s' and certificate in: '%s'\n", options.MineKey, options.MineCertificate)
		fmt.Printf("  generate-mine-cert PREFIX IPs... - create private key in: '<PREFIX>.key' certificate in: '<PREFIX>.crt'\n")
		exitwithstatus.Exit(1)
	}
}
