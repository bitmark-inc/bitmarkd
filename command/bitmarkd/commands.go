// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/sha3"

	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/mode"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
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
	case "gen-peer-identity", "peer":
		publicKeyFilename := options.Peering.PublicKey
		privateKeyFilename := options.Peering.PrivateKey

		if len(arguments) >= 1 && "" != arguments[0] {
			publicKeyFilename = arguments[0] + ".public"
			privateKeyFilename = arguments[0] + ".private"
		}
		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			log.Criticalf("cannot generate private key: %q and public key: %q", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %s\n", err)
			log.Criticalf("error generating server key pair: %s", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
		log.Infof("generated private key: %q and public key: %q", privateKeyFilename, publicKeyFilename)

	case "gen-rpc-cert", "rpc":
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
			fmt.Printf("error generating RPC key/certificate: %s\n", err)
			log.Criticalf("error generating RPC key/certificate: %s", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated RPC key: %q and certificate: %q\n", privateKeyFilename, certificateFilename)
		log.Infof("generated RPC key: %q and certificate: %q", privateKeyFilename, certificateFilename)

	case "gen-proof-identity", "proof":
		publicKeyFilename := options.Proofing.PublicKey
		privateKeyFilename := options.Proofing.PrivateKey
		signingKeyFilename := options.Proofing.SigningKey

		if len(arguments) >= 1 && "" != arguments[0] {
			publicKeyFilename = arguments[0] + ".public"
			privateKeyFilename = arguments[0] + ".private"
			signingKeyFilename = arguments[0] + ".sign"
		}
		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			log.Criticalf("cannot generate private key: %q and public key: %q", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %s\n", err)
			log.Criticalf("error generating server key pair: %s", err)
			exitwithstatus.Exit(1)
		}

		// new random seed for signing base record
		seedCore := make([]byte, 32)
		if _, err := rand.Read(seedCore); err != nil {
			fmt.Printf("error generating signing core error: %s\n", err)
			log.Criticalf("error generating signing core error: %s", err)
			exitwithstatus.Exit(1)
		}
		seed := []byte{0x5a, 0xfe, 0x01, 0x00} // header + network(live)
		if mode.IsTesting() {
			seed[3] = 0x01 // change network to testing
		}
		seed = append(seed, seedCore...)
		checksum := sha3.Sum256(seed)
		seed = append(seed, checksum[:4]...)

		data := "SEED:" + util.ToBase58(seed) + "\n"
		if err = ioutil.WriteFile(signingKeyFilename, []byte(data), 0600); err != nil {
			fmt.Printf("error writing signing key file error: %s\n", err)
			log.Criticalf("error writing signing key file error: %s", err)
			exitwithstatus.Exit(1)
		}

		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
		log.Infof("generated private key: %q and public key: %q", privateKeyFilename, publicKeyFilename)
		fmt.Printf("generated signing key: %q\n", signingKeyFilename)
		log.Infof("generated signing key: %q", signingKeyFilename)

	case "dns-txt", "txt":
		dnsTXT(log, options)

	case "start", "run":
		return false // continue processing

		// case "block-times":
		// 	return false // defer processing until database is loaded

	case "block", "b", "save-blocks", "save", "load-blocks", "load", "delete-down", "dd":
		return false // defer processing until database is loaded

	default:
		switch command {
		case "help", "h", "?":
		case "", " ":
			fmt.Printf("error: missing command\n")
		default:
			fmt.Printf("error: no such command: %q\n", command)
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

		fmt.Printf("  gen-proof-identity     (proof)   - create private key in: %q\n", options.Proofing.PrivateKey)
		fmt.Printf("                                     the public key in:     %q\n", options.Proofing.PublicKey)
		fmt.Printf("                                     and signing key in:    %q\n", options.Proofing.SigningKey)
		fmt.Printf("\n")

		fmt.Printf("  dns-txt                (txt)     - display the data to put in a dbs TXT record\n")
		fmt.Printf("\n")

		fmt.Printf("  start                  (run)     - just run the program, same as no arguments\n")
		fmt.Printf("                                     for convienience when passing script arguments\n")
		fmt.Printf("\n")

		fmt.Printf("  block S [E [FILE]]     (b)       - dump block(s) as a JSON structures to stdout/file\n")
		fmt.Printf("\n")

		fmt.Printf("  save-blocks FILE       (save)    - dump all blocks to a file\n")
		fmt.Printf("\n")

		fmt.Printf("  load-blocks FILE       (load)    - restore all blocks from a file\n")
		fmt.Printf("                                     only runs if database is deleted first\n")
		fmt.Printf("\n")

		fmt.Printf("  delete-down NUMBER     (dd)      - delete blocks in descending order\n")
		fmt.Printf("\n")

		//fmt.Printf("  block-times FILE BEGIN END       - write time and difficulty to text file for a range of blocks\n")
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

	case "start", "run":
		return false // continue processing

	// case "block-times":
	// 	if len(arguments) < 3 {
	// 		fmt.Printf("missing arguments arguments (use '' for stdout, and '0' for min/max)\n")
	// 		exitwithstatus.Exit(1)
	// 	}

	// 	begin, err := strconv.ParseUint(arguments[1], 10, 64)
	// 	if nil != err {
	// 		fmt.Printf("error in begin block number: %s\n", err)
	// 		exitwithstatus.Exit(1)
	// 	}
	// 	end, err := strconv.ParseUint(arguments[2], 10, 64)
	// 	if nil != err {
	// 		fmt.Printf("error in end block number: %s\n", err)
	// 		exitwithstatus.Exit(1)
	// 	}

	// 	fmt.Printf("*********** ERROR: %d %d\n", begin, end) // ***** FIX THIS: remove later when this code

	// 	switch filename := arguments[0]; filename {
	// 	case "": // use stdout
	// 		fallthrough
	// 	case "-": // use stdout
	// 		// block.PrintBlockTimes(os.Stdout, begin, end)
	// 		panic("HERE")

	// 	default:
	// 		fh, err := os.Create(filename)

	// 		if nil != err {
	// 			fmt.Printf("cannot create: %q  error: %s\n", filename, err)
	// 			exitwithstatus.Exit(1)
	// 		}
	// 		defer fh.Close()
	// 		//block.PrintBlockTimes(fh, begin, end)
	// 		panic("HERE")
	// 	}

	case "block", "b":
		if len(arguments) < 1 {
			exitwithstatus.Message("missing block number argument\n")
		}

		n, err := strconv.ParseUint(arguments[0], 10, 64)
		if nil != err {
			exitwithstatus.Message("error in block number: %s\n", err)
		}
		if n < 2 {
			exitwithstatus.Message("error: invalid block number: %d must be greater than 1\n", n)
		}

		output := "-"

		// optional end range
		nEnd := n
		if len(arguments) > 1 {

			nEnd, err = strconv.ParseUint(arguments[1], 10, 64)
			if nil != err {
				exitwithstatus.Message("error in ending block number: %s\n", err)
			}
			if nEnd < n {
				exitwithstatus.Message("error: invalid ending block number: %d must be greater than 1\n", n)
			}
		}

		if len(arguments) > 2 {
			output = strings.TrimSpace(arguments[2])
		}
		fd := os.Stdout

		if output != "" && output != "-" {
			fd, err = os.Create(output)
			if nil != err {
				exitwithstatus.Message("error: creating: %q error: %s", output, err)
			}
		}

		fmt.Fprintf(fd, "[\n")
		for ; n <= nEnd; n += 1 {
			block, err := dumpBlock(n)
			if nil != err {
				exitwithstatus.Message("dump block error: %s\n", err)
			}
			s, err := json.MarshalIndent(block, "  ", "  ")
			if nil != err {
				exitwithstatus.Message("dump block JSON error: %s\n", err)
			}

			fmt.Fprintf(fd, "  %s,\n", s)
		}
		fmt.Fprintf(fd, "{}]\n")
		fd.Close()

	case "save-blocks", "save":
		if len(arguments) < 1 {
			exitwithstatus.Message("missing file name argument\n")
		}
		filename := arguments[0]
		if "" == filename {
			exitwithstatus.Message("missing file name\n")
		}
		err := saveBinaryBlocks(filename)
		if nil != err {
			exitwithstatus.Message("failed writing: %q  error: %s\n", filename, err)
		}

	case "load-blocks", "load":
		if len(arguments) < 1 {
			exitwithstatus.Message("missing file name argument\n")
		}
		filename := arguments[0]
		if "" == filename {
			exitwithstatus.Message("missing file name\n")
		}
		err := restoreBinaryBlocks(filename)
		if nil != err {
			exitwithstatus.Message("failed writing: %q  error: %s\n", filename, err)
		}

	case "delete-down", "dd":
		// delete blocks down to a given block number
		if len(arguments) < 1 {
			exitwithstatus.Message("missing block number argument\n")
		}

		n, err := strconv.ParseUint(arguments[0], 10, 64)
		if nil != err {
			exitwithstatus.Message("error in block number: %s\n", err)
		}
		if n < 2 {
			exitwithstatus.Message("error: invalid block number: %d must be greater than 1\n", n)
		}
		err = block.DeleteDownToBlock(n)
		if nil != err {
			exitwithstatus.Message("block delete error: %s\n", err)
		}
		fmt.Printf("reduced height to: %d\n", block.GetHeight())

	default:
		exitwithstatus.Message("error: no such command: %s\n", command)

	}

	// indicate processing complete and perform normal exit from main
	return true
}

// print out the DNS TXT record
func dnsTXT(log *logger.L, options *Configuration) {
	//   <TAG> a=<IPv4;IPv6> c=<PEER-PORT> r=<RPC-PORT> f=<SHA3-256(cert)> p=<PUBLIC-KEY>
	const txtRecord = `TXT "bitmark=v3 a=%s c=%d r=%d f=%x p=%x"` + "\n"

	rpc := options.ClientRPC

	keypair, err := tls.LoadX509KeyPair(rpc.Certificate, rpc.PrivateKey)
	if nil != err {
		exitwithstatus.Message("error: cannot certificate: %q  error: %s\n", rpc.Certificate, err)
	}

	fingerprint := CertificateFingerprint(keypair.Certificate[0])

	rpcIP4, rpcIP6, rpcPort := getFirstConnections(rpc.Announce)
	if 0 == rpcPort {
		exitwithstatus.Message("error: cannot determine rpc port\n")
	}

	peering := options.Peering

	publicKey, err := zmqutil.ReadPublicKey(peering.PublicKey)
	if nil != err {
		exitwithstatus.Message("error: cannot read public key: %q  error: %s\n", peering.PublicKey, err)
	}

	peeringAnnounce := options.Peering.Announce

	listenIP4, listenIP6, listenPort := getFirstConnections(peeringAnnounce)
	if 0 == listenPort {
		exitwithstatus.Message("error: cannot determine listen port\n")
	}

	IPs := ""
	if "" != rpcIP4 && rpcIP4 == listenIP4 {
		IPs = rpcIP4
	}
	if "" != rpcIP6 && rpcIP6 == listenIP6 {
		if "" == IPs {
			IPs = rpcIP6
		} else {
			IPs += ";" + rpcIP6
		}
	}

	fmt.Printf("rpc fingerprint: %x\n", fingerprint)
	fmt.Printf("rpc port:        %d\n", rpcPort)
	fmt.Printf("public key:      %x\n", publicKey)
	fmt.Printf("connect port:    %d\n", listenPort)
	fmt.Printf("IP4 IP6:         %s\n", IPs)

	fmt.Printf(txtRecord, IPs, listenPort, rpcPort, fingerprint, publicKey)
}

// extract first IP4 and/or IP6 connection
func getFirstConnections(connections []string) (string, string, int) {
	initialPort := 0
	IP4 := ""
	IP6 := ""

scan_connections:
	for i, c := range connections {
		if "" == c {
			continue scan_connections
		}
		v6, IP, port, err := splitConnection(c)
		if nil != err {
			exitwithstatus.Message("error: cannot decode[%d]: %q  error: %s\n", i, c, err)
		}
		if v6 {
			if "" == IP6 {
				IP6 = IP
				if 0 == initialPort || port == initialPort {
					initialPort = port
				}
			}
		} else {
			if "" == IP4 {
				IP4 = IP
				if 0 == initialPort || port == initialPort {
					initialPort = port
				}
			}
		}
	}
	return IP4, IP6, initialPort
}

// split connection into ip and port
func splitConnection(hostPort string) (bool, string, int, error) {
	host, port, err := net.SplitHostPort(hostPort)
	if nil != err {
		return false, "", 0, fault.ErrInvalidIPAddress
	}

	IP := net.ParseIP(strings.Trim(host, " "))
	if nil == IP {
		return false, "", 0, fault.ErrInvalidIPAddress
	}

	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return false, "", 0, err
	}
	if numericPort < 1 || numericPort > 65535 {
		return false, "", 0, fault.ErrInvalidPortNumber
	}

	if nil != IP.To4() {
		return false, IP.String(), numericPort, nil
	}
	return true, "[" + IP.String() + "]", numericPort, nil
}
