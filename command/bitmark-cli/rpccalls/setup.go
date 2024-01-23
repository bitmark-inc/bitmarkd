// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
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

// Client - to hold RPC connections streams
type Client struct {
	conn    net.Conn
	client  *rpc.Client
	testnet bool
	verbose bool
	handle  io.Writer // if verbose is set output items here
}

// NewClient - create a RPC connection to a bitmarkd
func NewClient(testnet bool, connect string, verbose bool, handle io.Writer) (*Client, error) {

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	conn, err := tls.Dial("tcp", connect, tlsConfig)
	if err != nil {
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

// Close - shutdown the bitmarkd connection
func (c *Client) Close() {
	c.client.Close()
	c.conn.Close()
}
