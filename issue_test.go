// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"github.com/bitmark-inc/bitmarkd/rpc"
	"net/rpc/jsonrpc"
	"testing"
)

func TestConnect(t *testing.T) {
	conn, err := connect("node-1.test.bitmark.com:3130")

	if nil != err {
		t.Errorf("Connect failed: %v\n", err)
	}
	defer conn.Close()

	client := jsonrpc.NewClient(conn)
	defer client.Close()

	var reply rpc.InfoReply
	err = client.Call("Node.Info", nil, &reply)
	if nil != err {
		t.Errorf("Request info failed: %v\n", err)
	}
	fmt.Printf("Info: chain: %s, mode: %s\n", reply.Chain, reply.Mode)

	// make asset
	// err, assetIndex := makeAsset(client, registrant, rpc.testNet, asset, verbose)

}
