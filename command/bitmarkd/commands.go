// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	peerPublicKeyFilename  = "peer.public"
	peerPrivateKeyFilename = "peer.private"

	rpcCertificateKeyFilename = "rpc.crt"
	rpcPrivateKeyFilename     = "rpc.key"

	proofPublicKeyFilename      = "proof.public"
	proofPrivateKeyFilename     = "proof.private"
	proofLiveSigningKeyFilename = "proof.live"
	proofTestSigningKeyFilename = "proof.test"
)

// setup command handler
// commands that run to create key and certificate files
// these commands cannot access any internal database or states
func processSetupCommand(arguments []string) bool {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {
	case "gen-peer-identity", "peer":
		publicKeyFilename := getFilenameWithDirectory(arguments, peerPublicKeyFilename)
		privateKeyFilename := getFilenameWithDirectory(arguments, peerPrivateKeyFilename)
		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %s\n", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)

	case "gen-rpc-cert", "rpc":
		certificateFilename := getFilenameWithDirectory(arguments, rpcCertificateKeyFilename)
		privateKeyFilename := getFilenameWithDirectory(arguments, rpcPrivateKeyFilename)
		addresses := []string{}
		if len(arguments) >= 2 {
			for _, a := range arguments[1:] {
				if "" != a {
					addresses = append(addresses, a)
				}
			}
		}
		err := makeSelfSignedCertificate("rpc", certificateFilename, privateKeyFilename, 0 != len(addresses), addresses)
		if nil != err {
			fmt.Printf("cannot generate RPC key: %q and certificate: %q\n", privateKeyFilename, certificateFilename)
			fmt.Printf("error generating RPC key/certificate: %s\n", err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated RPC key: %q and certificate: %q\n", privateKeyFilename, certificateFilename)

	case "gen-proof-identity", "proof":
		publicKeyFilename := getFilenameWithDirectory(arguments, proofPublicKeyFilename)
		privateKeyFilename := getFilenameWithDirectory(arguments, proofPrivateKeyFilename)
		liveSigningKeyFilename := getFilenameWithDirectory(arguments, proofLiveSigningKeyFilename)
		testSigningKeyFilename := getFilenameWithDirectory(arguments, proofTestSigningKeyFilename)
		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("cannot generate private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
			fmt.Printf("error generating server key pair: %s\n", err)
			exitwithstatus.Exit(1)
		}

		if err := makeSigningKey(false, liveSigningKeyFilename); err != nil {
			fmt.Printf("cannot generate the signing key for livenet: %q\n", liveSigningKeyFilename)
			fmt.Printf("error generatingthe signing key for livenet: %s\n", err)
			exitwithstatus.Exit(1)
		}
		if err := makeSigningKey(true, testSigningKeyFilename); err != nil {
			fmt.Printf("cannot generate the signing key for testnet: %q\n", testSigningKeyFilename)
			fmt.Printf("error generatingthe signing key for testnet: %s\n", err)
			exitwithstatus.Exit(1)
		}

		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
		fmt.Printf("generated signing keys: %q and %q\n", liveSigningKeyFilename, testSigningKeyFilename)

	case "dns-txt", "txt":
		return false // defer processing until configuration is read

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
		fmt.Printf("  help                       (h)      - display this message\n\n")

		fmt.Printf("  gen-peer-identity [DIR]    (peer)   - create private key in: %q\n", "DIR/"+peerPrivateKeyFilename)
		fmt.Printf("                                        and the public key in: %q\n", "DIR/"+peerPublicKeyFilename)
		fmt.Printf("\n")

		fmt.Printf("  gen-rpc-cert [DIR]         (rpc)    - create private key in:  %q\n", "DIR/"+rpcPrivateKeyFilename)
		fmt.Printf("                                        and the certificate in: %q\n", "DIR/"+rpcCertificateKeyFilename)
		fmt.Printf("\n")

		fmt.Printf("  gen-rpc-cert [DIR] [IPs...]         - create private key in:  %q\n", "DIR/"+rpcPrivateKeyFilename)
		fmt.Printf("                                        and the certificate in: %q\n", "DIR/"+rpcCertificateKeyFilename)
		fmt.Printf("\n")

		fmt.Printf("  gen-proof-identity [DIR]   (proof)  - create private key in: %q\n", "DIR/"+proofPrivateKeyFilename)
		fmt.Printf("                                        the public key in:     %q\n", "DIR/"+proofPublicKeyFilename)
		fmt.Printf("                                        and signing keys in:  %q and: %q\n", "DIR/"+proofLiveSigningKeyFilename, "DIR/"+proofTestSigningKeyFilename)
		fmt.Printf("\n")

		fmt.Printf("  dns-txt                    (txt)    - display the data to put in a dbs TXT record\n")
		fmt.Printf("\n")

		fmt.Printf("  start                      (run)    - just run the program, same as no arguments\n")
		fmt.Printf("                                        for convienience when passing script arguments\n")
		fmt.Printf("\n")

		fmt.Printf("  block S [E [FILE]]         (b)      - dump block(s) as a JSON structures to stdout/file\n")
		fmt.Printf("\n")

		fmt.Printf("  save-blocks FILE           (save)   - dump all blocks to a file\n")
		fmt.Printf("\n")

		fmt.Printf("  load-blocks FILE           (load)   - restore all blocks from a file\n")
		fmt.Printf("                                        only runs if database is deleted first\n")
		fmt.Printf("\n")

		fmt.Printf("  delete-down NUMBER         (dd)     - delete blocks in descending order\n")
		fmt.Printf("\n")

		exitwithstatus.Exit(1)
	}

	// indicate processing complete and prefor normal exit from main
	return true
}

// configuration file enquiry commands
// have configuration file read and decoded, but nothing else
func processConfigCommand(arguments []string, options *Configuration) bool {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
		arguments = arguments[1:]
	}

	switch command {

	case "dns-txt", "txt":
		dnsTXT(options)

	default: // unknown commands fall through to data command
		return false
	}

	// indicate processing complete and perform normal exit from main
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

	case "block", "b":
		if len(arguments) < 1 {
			exitwithstatus.Message("missing block number argument")
		}

		n, err := strconv.ParseUint(arguments[0], 10, 64)
		if nil != err {
			exitwithstatus.Message("error in block number: %s", err)
		}
		if n < 2 {
			exitwithstatus.Message("error: invalid block number: %d must be greater than 1", n)
		}

		output := "-"

		// optional end range
		nEnd := n
		if len(arguments) > 1 {

			nEnd, err = strconv.ParseUint(arguments[1], 10, 64)
			if nil != err {
				exitwithstatus.Message("error in ending block number: %s", err)
			}
			if nEnd < n {
				exitwithstatus.Message("error: invalid ending block number: %d must be greater than 1", n)
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
				exitwithstatus.Message("dump block error: %s", err)
			}
			s, err := json.MarshalIndent(block, "  ", "  ")
			if nil != err {
				exitwithstatus.Message("dump block JSON error: %s", err)
			}

			fmt.Fprintf(fd, "  %s,\n", s)
		}
		fmt.Fprintf(fd, "{}]\n")
		fd.Close()

	case "save-blocks", "save":
		if len(arguments) < 1 {
			exitwithstatus.Message("missing file name argument")
		}
		filename := arguments[0]
		if "" == filename {
			exitwithstatus.Message("missing file name")
		}
		err := saveBinaryBlocks(filename)
		if nil != err {
			exitwithstatus.Message("failed writing: %q  error: %s", filename, err)
		}

	case "load-blocks", "load":
		if len(arguments) < 1 {
			exitwithstatus.Message("missing file name argument")
		}
		filename := arguments[0]
		if "" == filename {
			exitwithstatus.Message("missing file name")
		}
		err := restoreBinaryBlocks(filename)
		if nil != err {
			exitwithstatus.Message("failed writing: %q  error: %s", filename, err)
		}

	case "delete-down", "dd":
		// delete blocks down to a given block number
		if len(arguments) < 1 {
			exitwithstatus.Message("missing block number argument")
		}

		n, err := strconv.ParseUint(arguments[0], 10, 64)
		if nil != err {
			exitwithstatus.Message("error in block number: %s", err)
		}
		if n < 2 {
			exitwithstatus.Message("error: invalid block number: %d must be greater than 1", n)
		}
		err = block.DeleteDownToBlock(n)
		if nil != err {
			exitwithstatus.Message("block delete error: %s", err)
		}
		fmt.Printf("reduced height to: %d\n", blockheader.Height())

	default:
		exitwithstatus.Message("error: no such command: %s", command)

	}

	// indicate processing complete and perform normal exit from main
	return true
}

// print out the DNS TXT record
func dnsTXT(options *Configuration) {
	//   <TAG> a=<IPv4;IPv6> c=<PEER-PORT> r=<RPC-PORT> f=<SHA3-256(cert)> p=<PUBLIC-KEY>
	const txtRecord = `TXT "bitmark=v3 a=%s c=%d r=%d f=%x p=%x"` + "\n"

	rpc := options.ClientRPC

	keypair, err := tls.X509KeyPair([]byte(rpc.Certificate), []byte(rpc.PrivateKey))
	if nil != err {
		exitwithstatus.Message("error: cannot decode certificate: %q  error: %s", rpc.Certificate, err)
	}

	fingerprint := CertificateFingerprint(keypair.Certificate[0])

	if 0 == len(rpc.Announce) {
		exitwithstatus.Message("error: no rpc announce fields given")
	}

	rpcIP4, rpcIP6, rpcPort := getFirstConnections(rpc.Announce)
	if 0 == rpcPort {
		exitwithstatus.Message("error: cannot determine rpc port")
	}

	peering := options.Peering

	publicKey, err := zmqutil.ReadPublicKey(peering.PublicKey)
	if nil != err {
		exitwithstatus.Message("error: cannot read public key: %q  error: %s", peering.PublicKey, err)
	}

	if 0 == len(peering.Announce) {
		exitwithstatus.Message("error: no rpc announce fields given")
	}

	listenIP4, listenIP6, listenPort := getFirstConnections(peering.Announce)
	if 0 == listenPort {
		exitwithstatus.Message("error: cannot determine listen port")
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
			exitwithstatus.Message("error: cannot decode[%d]: %q  error: %s", i, c, err)
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

// get the working directory; if not set in the arguments
// it's set to the current directory
func getFilenameWithDirectory(arguments []string, name string) string {
	dir := "."
	if len(arguments) >= 1 {
		dir = arguments[0]
	}

	return filepath.Join(dir, name)
}

func makeSigningKey(test bool, fileName string) error {
	seedCore := make([]byte, 32)
	if _, err := rand.Read(seedCore); err != nil {
		return fmt.Errorf("error generating signing core error: %s\n", err)
	}
	seed := []byte{0x5a, 0xfe, 0x01, 0x00} // header + network(live)
	if test {
		seed[3] = 0x01 // change network to testing
	}
	seed = append(seed, seedCore...)
	checksum := sha3.Sum256(seed)
	seed = append(seed, checksum[:4]...)

	data := "SEED:" + util.ToBase58(seed) + "\n"
	if err := ioutil.WriteFile(fileName, []byte(data), 0600); err != nil {
		return fmt.Errorf("error writing signing key file error: %s\n", err)
	}

	return nil
}
