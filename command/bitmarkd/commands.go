// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bitmark-inc/bitmarkd/account"
	"github.com/bitmark-inc/bitmarkd/block"
	"github.com/bitmark-inc/bitmarkd/blockheader"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/util"
	"github.com/bitmark-inc/bitmarkd/zmqutil"
	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/logger"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
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
	case "gen-peer-identity", "peer":
		privateKeyFilename := getFilenameWithDirectory(arguments, peerPrivateKeyFilename)

		if util.EnsureFileExists(peerPrivateKeyFilename) {
			fmt.Printf("generate private key: %q error: %s\n", privateKeyFilename, fault.ErrCertificateFileAlreadyExists)
			exitwithstatus.Exit(1)
		}

		key, err := util.MakeEd25519PeerKey()
		if err != nil {
			fmt.Printf("generate private key: %q error: %s\n", privateKeyFilename, err.Error())
			exitwithstatus.Exit(1)
		}

		if err := ioutil.WriteFile(privateKeyFilename, []byte(key), 0600); err != nil {
			os.Remove(privateKeyFilename)
			fmt.Printf("generate private key: %q error: %s\n", privateKeyFilename, err.Error())
			exitwithstatus.Exit(1)
		}

		fmt.Printf("generated private key: %q\n", privateKeyFilename)

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
			fmt.Printf("generate RPC key: %q and certificate: %q error: %s\n", privateKeyFilename, certificateFilename, err)
			exitwithstatus.Exit(1)
		}
		fmt.Printf("generated RPC key: %q and certificate: %q\n", privateKeyFilename, certificateFilename)

	case "gen-proof-identity", "proof":
		publicKeyFilename := getFilenameWithDirectory(arguments, proofPublicKeyFilename)
		privateKeyFilename := getFilenameWithDirectory(arguments, proofPrivateKeyFilename)
		err := zmqutil.MakeKeyPair(publicKeyFilename, privateKeyFilename)
		if nil != err {
			fmt.Printf("generate private key: %q and public key: %q error: %s\n", privateKeyFilename, publicKeyFilename, err)
			exitwithstatus.Exit(1)
		}

		liveSigningKeyFilename := getFilenameWithDirectory(arguments, proofLiveSigningKeyFilename)
		testSigningKeyFilename := getFilenameWithDirectory(arguments, proofTestSigningKeyFilename)

		if err := makeSigningKey(false, liveSigningKeyFilename); nil != err {
			fmt.Printf("generate the signing key for livenet: %q error: %s\n", liveSigningKeyFilename, err)
			goto signing_key_failed
		}
		if err := makeSigningKey(true, testSigningKeyFilename); nil != err {
			fmt.Printf(" generate the signing key for testnet: %q error: %s\n", testSigningKeyFilename, err)
			goto signing_key_failed
		}

		fmt.Printf("generated private key: %q and public key: %q\n", privateKeyFilename, publicKeyFilename)
		fmt.Printf("generated signing keys: %q and %q\n", liveSigningKeyFilename, testSigningKeyFilename)
		return true

	signing_key_failed:
		_ = os.Remove(publicKeyFilename)
		_ = os.Remove(privateKeyFilename)
		_ = os.Remove(liveSigningKeyFilename)
		_ = os.Remove(testSigningKeyFilename)
		exitwithstatus.Exit(1)

	case "dns-txt", "txt":
		return false // defer processing until configuration is read

	case "start", "run":
		return false // continue processing

	case "block", "b", "save-blocks", "save", "load-blocks", "load", "delete-down", "dd":
		return false // defer processing until database is loaded

	case "config-test", "cfg":
		return false

	case "version", "v":
		fmt.Printf("%s\n", version)
		return true

	default:
		switch command {
		case "help", "h", "?":
		case "", " ":
			fmt.Printf("error: missing command\n")
		default:
			fmt.Printf("error: no such command: %q\n", command)
		}
		fmt.Printf("usage: %s [--help] [--verbose] [--quiet] --config-file=FILE [[command|help] arguments...]", program)

		fmt.Printf("supported commands:\n\n")
		fmt.Printf("  help                       (h)      - display this message\n\n")
		fmt.Printf("  version                    (v)      - display version sting\n\n")

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

		fmt.Printf("  config-test                (cfg)    - just check the configuration file\n")
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

	// indicate processing complete and preform normal exit from main
	return true
}

// configuration file enquiry commands
// have configuration file read and decoded, but nothing else
func processConfigCommand(arguments []string, options *Configuration) bool {

	command := "help"
	if len(arguments) > 0 {
		command = arguments[0]
	}

	switch command {
	case "dns-txt", "txt":
		dnsTXT(options)

	case "config-test", "cfg":
		b, err := json.Marshal(options)
		if err != nil {
			exitwithstatus.Message("error: %s", err)
		}
		var out bytes.Buffer
		json.Indent(&out, b, "", "  ")
		out.WriteTo(os.Stdout)
		os.Stdout.WriteString("\n")

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
	const txtRecord = `TXT "bitmark=v3 a=%s c=%d r=%d f=%x i=%s"` + "\n"

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

	privateKeyBytes, err := hex.DecodeString(peering.PrivateKey)
	if err != nil {
		exitwithstatus.Message("error: cannot decode private key: %q  error: %s", peering.PrivateKey, err)
	}

	privateKey, err := crypto.UnmarshalPrivateKey(privateKeyBytes)
	if err != nil {
		exitwithstatus.Message("error: cannot generate private key: %q  error: %s", peering.PrivateKey, err)
	}

	peerID, err := peer.IDFromPrivateKey(privateKey)
	if err != nil {
		exitwithstatus.Message("error: cannot generate peer id  error: %s", err)
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
	fmt.Printf("connect port:    %d\n", listenPort)
	fmt.Printf("peer id:         %s\n", peerID)
	fmt.Printf("IP4 IP6:         %s\n", IPs)

	fmt.Printf(txtRecord, IPs, listenPort, rpcPort, fingerprint, peerID)
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
		return false, "", 0, fault.InvalidIpAddress
	}

	IP := net.ParseIP(strings.Trim(host, " "))
	if nil == IP {
		return false, "", 0, fault.InvalidIpAddress
	}

	numericPort, err := strconv.Atoi(strings.Trim(port, " "))
	if nil != err {
		return false, "", 0, err
	}
	if numericPort < 1 || numericPort > 65535 {
		return false, "", 0, fault.InvalidPortNumber
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

func makeSigningKey(testnet bool, fileName string) error {
	seed, err := account.NewBase58EncodedSeedV2(testnet)
	if nil != err {
		return err
	}

	data := "SEED:" + seed + "\n"
	if err = ioutil.WriteFile(fileName, []byte(data), 0600); nil != err {
		return fmt.Errorf("error writing signing key file error: %s", err)
	}

	return nil
}
