// SPDX-License-Identifier: ISC
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

	"github.com/bitmark-inc/exitwithstatus"
	"github.com/bitmark-inc/getoptions"
)

// RPCEmptyArguments - null parameters for RPC call
type RPCEmptyArguments struct{}

// RPCClient - client connection for RPC calls
type RPCClient struct {
	Client *rpc.Client
}

// GetNodeInfo - returns the node info of a node from bitmark rpc
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
		// {Long: "verbose", HasArg: getoptions.NO_ARGUMENT, Short: 'v'},
		// {Long: "quiet", HasArg: getoptions.NO_ARGUMENT, Short: 'q'},
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
		exitwithstatus.Message("usage: %s [--help] host:port", program)
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

	reply := map[string]interface{}{
		"host": fmt.Sprintf("tcp://%s", hostPort),
	}

	reply["info"], err = r.GetNodeInfo()

	if err != nil {
		exitwithstatus.Message("rpc error: %s", err)
	}

	b, err := json.Marshal(reply)
	if err != nil {
		exitwithstatus.Message("incorrect json marshal: %s", err)
	}

	fmt.Printf("%s", b)
}
