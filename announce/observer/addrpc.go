// SPDX-License-Identifier: ISC
// Copyright (c) 2014-2020 Bitmark Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package observer

import (
	"encoding/binary"

	"github.com/bitmark-inc/bitmarkd/announce/rpc"
	"github.com/bitmark-inc/logger"
)

const (
	addrpcEvent = "addrpc"
)

type addrpc struct {
	rpcs rpc.RPC
	log  *logger.L
}

func (a addrpc) Update(str string, args [][]byte) {
	if str == addrpcEvent {
		timestamp := binary.BigEndian.Uint64(args[2])
		a.log.Infof("received rpc: fingerprint: %x  rpc: %x  Timestamp: %d", args[0], args[1], timestamp)
		a.rpcs.Add(args[0], args[1], timestamp)
	}
}

func NewAddrpc(rpcs rpc.RPC, log *logger.L) Observer {
	return &addrpc{rpcs: rpcs, log: log}
}
