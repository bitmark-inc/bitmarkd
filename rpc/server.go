// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2019 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"

	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/logger"
)

// global atomic connection counter
// all listening ports share this count
var connectionCountRPC counter.Counter

// a single socket RPC listener
func listenAndServeRPC(listen net.Listener, server *rpc.Server, maximumConnections uint64, log *logger.L) {
accept_loop:
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Errorf("rpc.Server terminated: accept error:", err)
			break accept_loop
		}
		if connectionCountRPC.Increment() <= maximumConnections {
			go func() {
				server.ServeCodec(jsonrpc.NewServerCodec(conn))
				conn.Close()
				connectionCountRPC.Decrement()
			}()
		} else {
			connectionCountRPC.Decrement()
			conn.Close()
		}

	}
	listen.Close()
	log.Error("RPC accept terminated")
}
