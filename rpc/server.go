// Copyright (c) 2014-2016 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/bitmark-inc/bitmarkd/counter"
	"github.com/bitmark-inc/logger"
	"io"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"
)

// limit the number of gets
const MaximumGetSize = 100

// the argument passed to the callback
type ServerArgument struct {
	Log       *logger.L
	StartTime time.Time
}

var connectionCount counter.Counter

// listener callback
func Callback(conn io.ReadWriteCloser, argument interface{}) {

	serverArgument := argument.(*ServerArgument)
	if nil == serverArgument {
		panic("rpc: nil serverArgument")
	}
	if nil == serverArgument.Log {
		panic("rpc: nil serverArgument.Log")
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

	owner := &Owner{
		log: serverArgument.Log,
	}

	blk := &Block{
		log: serverArgument.Log,
	}

	tx := &Transaction{
		log: serverArgument.Log,
	}

	pool := &Pool{
		log: serverArgument.Log,
	}

	node := &Node{
		log:   serverArgument.Log,
		start: serverArgument.StartTime,
	}

	server := rpc.NewServer()
	server.Register(asset)
	server.Register(assets)
	server.Register(bitmark)
	server.Register(bitmarks)
	server.Register(owner)
	server.Register(blk)
	server.Register(tx)
	server.Register(pool)
	server.Register(node)

	connectionCount.Increment()
	defer connectionCount.Decrement()

	codec := jsonrpc.NewServerCodec(conn)
	defer codec.Close()
	server.ServeCodec(codec)
}
