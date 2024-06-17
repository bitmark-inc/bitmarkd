// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
)

type RPCEmptyArguments struct{}

type Connected struct {
	Address string `json:"address"`
	Server  string `json:"server"`
}

type ConnClient struct {
	Clients []Connected `json:"clients"`
}

type RPCClient struct {
	Client *rpc.Client
}

// GetNodeInfo will get the node info of a node from bitmark rpc
func (r *RPCClient) GetNodeInfo() (json.RawMessage, error) {
	args := RPCEmptyArguments{}
	var reply json.RawMessage
	err := r.Client.Call("Node.Info", &args, &reply)
	return reply, err
}

// set by the linker: go build -ldflags "-X main.version=M.N" ./...
var version = "zero" // do not change this value

// main program
func main() {
	defer exitwithstatus.Handler()

	flags := []getoptions.Option{
		{Long: "help", HasArg: getoptions.NO_ARGUMENT, Short: 'h'},
		{Long: "time", HasArg: getoptions.REQUIRED_ARGUMENT, Short: 't'},
		{Long: "verbose", HasArg: getoptions.NO_ARGUMENT, Short: 'v'},
		{Long: "quiet", HasArg: getoptions.NO_ARGUMENT, Short: 'q'},
		{Long: "version", HasArg: getoptions.NO_ARGUMENT, Short: 'V'},
	}

	program, options, arguments, err := getoptions.GetOS(flags)
	if err != nil {
		exitwithstatus.Message("option parse error: %s", err)
	}

	if len(options["version"]) > 0 {
		exitwithstatus.Message("%s: version: %s", program, version)
	}

	if len(options["help"]) > 0 || len(arguments) == 0 {
		exitwithstatus.Message("usage: %s [--help] [--time=N{h|m|s}] [host:port]", program)
	}

	verbose := len(options["verbose"]) > 0
	quiet := len(options["quiet"]) > 0

	sampleTime := time.Minute
	if len(options["time"]) > 0 {
		sampleTime, err = time.ParseDuration(options["time"][0])
		if err != nil {
			exitwithstatus.Message("%s: convert time error: %s", program, err)
		}
		if sampleTime.Seconds() < 1 {
			exitwithstatus.Message("%s: invalid time: %d", program, sampleTime)
		}
	}

	if len(arguments) != 1 {
		exitwithstatus.Message("%s: extraneous extra arguments", program)
	}
	hostPort := arguments[0]

	// establish rpc connection over tls
	conn, err := tls.Dial("tcp", hostPort, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		exitwithstatus.Message("dial error: %s", err)
	}
	defer conn.Close()
	client := jsonrpc.NewClient(conn)

	r := RPCClient{client}

	if !quiet {
		fmt.Printf("sending requests for: %7.1f seconds\n", sampleTime.Seconds())
	}

	total := 0
	end := time.Now().Add(sampleTime)
	for time.Now().Before(end) {

		reply, err := r.GetNodeInfo()
		if err != nil {
			exitwithstatus.Message("rpc error: %s", err)
		}
		total += 1
		b, err := json.Marshal(reply)
		if err != nil {
			exitwithstatus.Message("incorrect json marshal: %s", err)
		}

		if verbose {
			fmt.Printf("%s", b)
		}
	}

	if !quiet {
		fmt.Printf("finished\n")
	}

	fmt.Printf("total: %8d   requests in: %7.1f seconds\n", total, sampleTime.Seconds())
	fmt.Printf("rate:  %10.1f requests/second\n", float64(total)/sampleTime.Seconds())
}
