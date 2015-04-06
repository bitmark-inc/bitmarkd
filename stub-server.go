// Copyright (c) 2014-2015 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"github.com/bitmark-inc/listener"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
)

// simple type with state

/* examples
{"method":"One.Inc","id":1,"params":[{"Delta":1}]}
{"method":"One.Inc","id":2,"params":[{"Delta":24}]}
{"method":"One.Inc","id":3,"params":[{"Delta":-15}]}
{"method":"One.Inc","id":4,"params":[{"Delta":"bad value"}]}

{"method":"Arith.Multiply","id":5,"params":[{"A":132,"B":12}]}
{"method":"Arith.Divide","id":6,"params":[{"A":1171,"B":137}]}

*/

type One struct {
	m     sync.Mutex
	value int
}

type OneArguments struct {
	Delta int
}

type OneReply struct {
	Before int
	After  int
}

func (t *One) Inc(arguments *OneArguments, reply *OneReply) error {
	t.m.Lock()
	defer t.m.Unlock()
	reply.Before = t.value
	t.value += arguments.Delta
	reply.After = t.value
	return nil
}

// tha Arith type

type Arguments struct {
	A int
	B int
}

type Quotient struct {
	Quotient  int
	Remainder int
}

type Arith int

func (t *Arith) Multiply(arguments *Arguments, reply *int) error {
	*reply = arguments.A * arguments.B
	return nil
}

func (t *Arith) Divide(arguments *Arguments, quo *Quotient) error {
	if arguments.B == 0 {
		return errors.New("divide by zero")
	}
	quo.Quotient = arguments.A / arguments.B
	quo.Remainder = arguments.A % arguments.B
	return nil
}

// listener callback
func StubCallback(conn *listener.ClientConnection, argument interface{}) {

	one := new(One)
	arith := new(Arith)

	server := rpc.NewServer()
	server.Register(one)
	server.Register(arith)

	codec := jsonrpc.NewServerCodec(conn)
	defer codec.Close()
	server.ServeCodec(codec)
}
