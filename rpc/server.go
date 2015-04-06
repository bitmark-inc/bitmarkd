// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/listener"
	"github.com/bitmark-inc/logger"
	"net/rpc"
	"net/rpc/jsonrpc"
)

// limit the number of gets
const MaximumGetSize = 50

// the argument passed to the callback
type ServerArgument struct {
	Log *logger.L
}

// listener callback
func Callback(conn *listener.ClientConnection, argument interface{}) {

	serverArgument := argument.(*ServerArgument)
	if nil == serverArgument {
		panic("rpc: nil serverArgument")
	}
	if nil == serverArgument.Log {
		panic("rpc: nil serverArgument.Log ")
	}

	asset := &Asset{
		log: serverArgument.Log,
	}

	assets := &Assets{
		log:   serverArgument.Log,
		asset: asset,
	}

	bitmark := &Bitmark{
		log: serverArgument.Log,
	}

	bitmarks := &Bitmarks{
		log:     serverArgument.Log,
		bitmark: bitmark,
	}

	tx := &Transaction{
		log: serverArgument.Log,
	}

	blk := &Block{
		log: serverArgument.Log,
	}

	node := &Node{
		log: serverArgument.Log,
	}

	server := rpc.NewServer()
	server.Register(asset)
	server.Register(assets)
	server.Register(bitmark)
	server.Register(bitmarks)
	server.Register(node)
	server.Register(blk)
	server.Register(tx)

	codec := jsonrpc.NewServerCodec(conn)
	defer codec.Close()
	server.ServeCodec(codec)
}
