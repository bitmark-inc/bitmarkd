// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	//"bytes"
	//"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"github.com/agl/ed25519"
	"github.com/bitmark-inc/bitmarkd/configuration"
	"github.com/bitmark-inc/bitmarkd/fault"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"github.com/bitmark-inc/bitmarkd/transaction"
	"github.com/bitmark-inc/exitwithstatus"
	"io/ioutil"
	"math"
	"net"
	netrpc "net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// a dummy signature to begin
var dummySignature transaction.Signature

// to hold a keypair for testing
type keyPair struct {
	publicKey  [32]byte
	privateKey [64]byte
}

// public/private keys

var registrant = keyPair{
	publicKey: [...]byte{
		0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2, 0x35,
		0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73, 0xf6,
		0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49, 0x7e,
		0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11, 0x01,
	},
	privateKey: [...]byte{
		0x66, 0xf5, 0x28, 0xd0, 0x2a, 0x64, 0x97, 0x3a,
		0x2d, 0xa6, 0x5d, 0xb0, 0x53, 0xea, 0xd0, 0xfd,
		0x94, 0xca, 0x93, 0xeb, 0x9f, 0x74, 0x02, 0x3e,
		0xbe, 0xdb, 0x2e, 0x57, 0xb2, 0x79, 0xfd, 0xf3,
		0x7a, 0x81, 0x92, 0x56, 0x5e, 0x6c, 0xa2, 0x35,
		0x80, 0xe1, 0x81, 0x59, 0xef, 0x30, 0x73, 0xf6,
		0xe2, 0xfb, 0x8e, 0x7e, 0x9d, 0x31, 0x49, 0x7e,
		0x79, 0xd7, 0x73, 0x1b, 0xa3, 0x74, 0x11, 0x01,
	},
}

var issuer = keyPair{
	publicKey: [...]byte{
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	},
	privateKey: [...]byte{
		0xf3, 0xf7, 0xa1, 0xfc, 0x33, 0x10, 0x71, 0xc2,
		0xb1, 0xcb, 0xbe, 0x4f, 0x3a, 0xee, 0x23, 0x5a,
		0xae, 0xcc, 0xd8, 0x5d, 0x2a, 0x80, 0x4c, 0x44,
		0xb5, 0xc6, 0x03, 0xb4, 0xca, 0x4d, 0x9e, 0xc0,
		0x9f, 0xc4, 0x86, 0xa2, 0x53, 0x4f, 0x17, 0xe3,
		0x67, 0x07, 0xfa, 0x4b, 0x95, 0x3e, 0x3b, 0x34,
		0x00, 0xe2, 0x72, 0x9f, 0x65, 0x61, 0x16, 0xdd,
		0x7b, 0x01, 0x8d, 0xf3, 0x46, 0x98, 0xbd, 0xc2,
	},
}

var ownerOne = keyPair{
	publicKey: [...]byte{
		0x27, 0x64, 0x0e, 0x4a, 0xab, 0x92, 0xd8, 0x7b,
		0x4a, 0x6a, 0x2f, 0x30, 0xb8, 0x81, 0xf4, 0x49,
		0x29, 0xf8, 0x66, 0x04, 0x3a, 0x84, 0x1c, 0x38,
		0x14, 0xb1, 0x66, 0xb8, 0x89, 0x44, 0xb0, 0x92,
	},
	privateKey: [...]byte{
		0xc7, 0xae, 0x9f, 0x22, 0x32, 0x0e, 0xda, 0x65,
		0x02, 0x89, 0xf2, 0x64, 0x7b, 0xc3, 0xa4, 0x4f,
		0xfa, 0xe0, 0x55, 0x79, 0xcb, 0x6a, 0x42, 0x20,
		0x90, 0xb4, 0x59, 0xb3, 0x17, 0xed, 0xf4, 0xa1,
		0x27, 0x64, 0x0e, 0x4a, 0xab, 0x92, 0xd8, 0x7b,
		0x4a, 0x6a, 0x2f, 0x30, 0xb8, 0x81, 0xf4, 0x49,
		0x29, 0xf8, 0x66, 0x04, 0x3a, 0x84, 0x1c, 0x38,
		0x14, 0xb1, 0x66, 0xb8, 0x89, 0x44, 0xb0, 0x92,
	},
}

var ownerTwo = keyPair{
	publicKey: [...]byte{
		0xa1, 0x36, 0x32, 0xd5, 0x42, 0x5a, 0xed, 0x3a,
		0x6b, 0x62, 0xe2, 0xbb, 0x6d, 0xe4, 0xc9, 0x59,
		0x48, 0x41, 0xc1, 0x5b, 0x70, 0x15, 0x69, 0xec,
		0x99, 0x99, 0xdc, 0x20, 0x1c, 0x35, 0xf7, 0xb3,
	},
	privateKey: [...]byte{
		0x8f, 0x83, 0x3e, 0x58, 0x30, 0xde, 0x63, 0x77,
		0x89, 0x4a, 0x8d, 0xf2, 0xd4, 0x4b, 0x17, 0x88,
		0x39, 0x1d, 0xcd, 0xb8, 0xfa, 0x57, 0x22, 0x73,
		0xd6, 0x2e, 0x9f, 0xcb, 0x37, 0x20, 0x2a, 0xb9,
		0xa1, 0x36, 0x32, 0xd5, 0x42, 0x5a, 0xed, 0x3a,
		0x6b, 0x62, 0xe2, 0xbb, 0x6d, 0xe4, 0xc9, 0x59,
		0x48, 0x41, 0xc1, 0x5b, 0x70, 0x15, 0x69, 0xec,
		0x99, 0x99, 0xdc, 0x20, 0x1c, 0x35, 0xf7, 0xb3,
	},
}

// the main program
func main() {
	// ensure exit handler is first
	defer exitwithstatus.Handler()

	// read options and parse the configuration file
	// also sets up and starts logging
	options := configuration.ParseOptions()

	if options.Version {
		exitwithstatus.Usage("Version: %s\n", Version())
	}

	if options.Verbose {
		fmt.Printf("options: %#v\n", options)
	}

	if len(options.RPCAnnounce) < 1 {
		exitwithstatus.Usage("there were no RpcAnnounce configuration values\n")
	}

	cerfificateFile, ok := configuration.ResolveFileName(options.RPCCertificate)
	if !ok {
		exitwithstatus.Usage("Certificate file: %q not found\n", cerfificateFile)
	}

	// connnect to first announced RPC port
	//conn := connect(cerfificateFile, options.RPCAnnounce[0])
	//defer conn.Close()

	// force the connection to localhost but with port from 1st rpc announce
	// 10.0.0.1:1234 or [fe80::f279:59ff:fe6a:474]:1234
	s := strings.Split(options.RPCAnnounce[0], ":")
	port := s[len(s)-1] // port is last element
	hostPort := "127.0.0.1:" + port
	if '[' == s[0][0] {
		hostPort = "[::1]:" + port
	}

	// process the command and its arguments
	command := options.Args.Command
	args := options.Args.Arguments
	switch command {

	case "rate":
		if 2 != len(args) {
			exitwithstatus.Usage("rate command needs rate(tx/s) and timeLimit(minutes): arg count: %d\n", len(args))
		}

		rateLimit, err := strconv.ParseFloat(args[0], 64)
		if nil != err {
			exitwithstatus.Usage("command invalid rate: %s  error: %v\n", args[0], err)
		}

		timeLimit, err := strconv.ParseFloat(args[1], 64)
		if nil != err {
			exitwithstatus.Usage("command invalid timeLimit: %s  error: %v\n", args[1], err)
		}

		if rateLimit <= 0 || timeLimit <= 0 {
			exitwithstatus.Usage("rate command invalid parameters, rate %6.2f (tx/s) and timeLimit %6.2f (minutes)\n", rateLimit, timeLimit)
		}
		process_rate(hostPort, cerfificateFile, rateLimit, timeLimit, options.Verbose)

	case "one":
		if 1 != len(args) {
			exitwithstatus.Usage("one command needs name: arg count: %d\n", len(args))
		}
		process_one(hostPort, cerfificateFile, args[0], options.Verbose)

	default:
		switch command {
		case "help", "h", "?":
		case "", " ":
			fmt.Printf("error: missing command\n")
		default:
			fmt.Printf("error: no such command: %v\n", command)
		}
		fmt.Printf("supported commands:\n")
		fmt.Printf("  help                                 - display this message\n")
		fmt.Printf("  rate tx/second time-limit(minutes)   - send one asset and a stream of issues for this asset\n")
		fmt.Printf("  one name                             - send one asset and one issue with alternative name (same fingerprint)\n")
		exitwithstatus.Exit(1)
	}

}

func process_one(hostPort string, cerfificateFile string, name string, verbose bool) {

	conn := connect(cerfificateFile, hostPort)
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	assetIndex := makeAsset(client, name, verbose)
	if nil == assetIndex {
		exitwithstatus.Usage("unable to get asset index\n")
	}

	doIssues(client, assetIndex, 1, verbose)
}

func process_rate(hostPort string, cerfificateFile string, rateLimit float64, timeLimit float64, verbose bool) {

	conn := connect(cerfificateFile, hostPort)
	defer conn.Close()

	// create a client
	client := jsonrpc.NewClient(conn)
	defer client.Close()

	assetIndex := makeAsset(client, "Item's Name", verbose)
	if nil == assetIndex {
		exitwithstatus.Usage("unable to get asset index\n")
	}

	// turn Signals into channel messages
	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	startTime := time.Now()
	counter := 0

	itemsPerCall := 1
	delay := time.Second

	if rateLimit > 1.0 {
		itemsPerCall = int(math.Ceil(rateLimit))
	} else {
		delay = time.Duration(1000.0/rateLimit) * time.Millisecond
	}
	if delay <= 0 {
		delay = time.Millisecond
	}

	timeout := time.After(time.Duration(timeLimit) * time.Minute)

	// send out until stopped
loop:
	for {
		doIssues(client, assetIndex, itemsPerCall, verbose)

		// compute block rate
		counter += itemsPerCall
		t := time.Since(startTime).Seconds()

		mm := int(math.Floor(t)) / 60
		ss := int(math.Floor(t)) % 60

		rate := float64(counter) / t
		r := rate
		if rate > 9999.99 {
			r = 9999.99
		}
		fmt.Printf("%02d:%02d  tx: %8d  rate: %7.2f tx/s\r", mm, ss, counter, r)

		if rate > rateLimit {
			select {
			case <-stop:
				break loop
			case <-timeout:
				break loop
			case <-time.After(delay): // rate limit
			}
		} else {
			select {
			case <-stop:
				break loop
			case <-timeout:
				break loop
			default:
			}
		}
	}
}

func doIssues(client *netrpc.Client, assetIndex *transaction.AssetIndex, issueCount int, verbose bool) {

	nonce := time.Now().UTC().Unix() * 1000
	issues := make([]*transaction.BitmarkIssue, issueCount)
	for i := 0; i < len(issues); i += 1 {
		issues[i] = makeIssue(assetIndex, uint64(nonce)+uint64(i))
	}

	if verbose {
		b, err := json.MarshalIndent(issues, "", "  ")
		if nil != err {
			fmt.Printf("json error: %v\n", err)
			return
		}

		fmt.Printf("JSON request:\n%s\n", b)
	}

	var reply []rpc.BitmarkIssueReply
	err := client.Call("Bitmarks.Issue", issues, &reply)
	if err != nil {
		fmt.Printf("Bitmark.Issue error: %v\n", err)
		return
	}

	if verbose {
		b, err := json.MarshalIndent(reply, "", "  ")
		if nil != err {
			fmt.Printf("json error: %v\n", err)
			return
		}

		fmt.Printf("JSON reply:\n%s\n", b)
	}

}

// helper to make an address
func makeAddress(publicKey *[32]byte) *transaction.Address {
	return &transaction.Address{
		AddressInterface: &transaction.ED25519Address{
			Test:      true,
			PublicKey: publicKey,
		},
	}
}

// build a properly signed asset
func makeAsset(client *netrpc.Client, name string, verbose bool) *transaction.AssetIndex {

	registrantAddress := makeAddress(&registrant.publicKey)

	r := transaction.AssetData{
		Description: "Just the description",
		Name:        name,
		Fingerprint: "0123456789abcdef",
		Registrant:  registrantAddress,
		Signature:   dummySignature,
	}

	packed, err := r.Pack(registrantAddress)
	if fault.ErrInvalidSignature != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(&registrant.privateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	packed, err = r.Pack(registrantAddress)
	if nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}

	if verbose {
		b, err := json.MarshalIndent(r, "", "  ")
		if nil != err {
			fmt.Printf("json error: %v\n", err)
			return nil
		}

		fmt.Printf("JSON request:\n%s\n", b)
	}

	var reply rpc.AssetRegisterReply
	err = client.Call("Asset.Register", r, &reply)
	if err != nil {
		fmt.Printf("Asset.Register error: %v\n", err)
		return nil
	}

	if verbose {
		b, err := json.MarshalIndent(reply, "", "  ")
		if nil != err {
			fmt.Printf("json error: %v\n", err)
			return nil
		}

		fmt.Printf("JSON REPLY:\n%s\n", b)
	}

	return &reply.AssetIndex
}

// build a properly signed issues
func makeIssue(assetIndex *transaction.AssetIndex, nonce uint64) *transaction.BitmarkIssue {

	issuerAddress := makeAddress(&issuer.publicKey)

	r := transaction.BitmarkIssue{
		AssetIndex: *assetIndex,
		Owner:      issuerAddress,
		Nonce:      nonce,
		Signature:  dummySignature,
	}

	packed, err := r.Pack(issuerAddress)
	if fault.ErrInvalidSignature != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}

	// manually sign the record and attach signature
	signature := ed25519.Sign(&issuer.privateKey, packed)
	r.Signature = signature[:]

	// re-pack with correct signature
	packed, err = r.Pack(issuerAddress)
	if nil != err {
		fmt.Printf("pack error: %v\n", err)
		return nil
	}
	return &r
}

// connect to bitmarkd RPC
func connect(certificateFileName string, connect string) net.Conn {

	_, err := os.Stat(certificateFileName)
	if nil != err {
		panic(fmt.Sprintf("certificate: %q does not exist\n", certificateFileName))
	}

	pemData, err := ioutil.ReadFile(certificateFileName)
	if nil != err {
		panic(fmt.Sprintf("certificate: %q cannot be read\n", certificateFileName))
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(pemData))
	if !ok {
		panic(fmt.Sprintf("failed to parse certificate file: %q", certificateFileName))
	}

	conn, err := tls.Dial("tcp", connect, &tls.Config{
		RootCAs: roots,
	})
	if nil != err {
		panic("failed to connect: " + err.Error())
	}

	return conn
}
