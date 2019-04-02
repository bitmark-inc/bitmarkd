// Copyright (c) 2014-2018 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpccalls

import (
	"crypto/tls"
	"io"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// to hold RPC connections streams
type Client struct {
	conn    net.Conn
	client  *rpc.Client
	testnet bool
	verbose bool
	handle  io.Writer // if verbose is set output items here
}

func NewClient(testnet bool, connect string, verbose bool, handle io.Writer) (*Client, error) {

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", connect, tlsConfig)
	if nil != err {
		return nil, err
	}

	r := &Client{
		conn:    conn,
		client:  jsonrpc.NewClient(conn),
		testnet: testnet,
		verbose: verbose,
		handle:  handle,
	}
	return r, nil
}

func (c *Client) Close() {
	c.client.Close()
	c.conn.Close()
}
