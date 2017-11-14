// Copyright (c) 2014-2017 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/logger"
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// limit the number of gets
//const MaximumGetSize = 100

// the argument passed to the callback
type serverArgument struct {
	Log    *logger.L
	Server *rpc.Server
}

var connectionCount counter.Counter

// listener callback
func Callback(conn io.ReadWriteCloser, argument interface{}) {

	serverArgument := argument.(*serverArgument)

	log := serverArgument.Log
	log.Info("starting…")

	server := serverArgument.Server

	connectionCount.Increment()
	defer connectionCount.Decrement()

	codec := jsonrpc.NewServerCodec(conn)
	defer codec.Close()
	server.ServeCodec(codec)

	log.Info("finished")
}
